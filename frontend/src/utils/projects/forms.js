// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { extractErrorMessage } from "@/utils/errors";
import { fieldLabel } from "@/utils/projects/rows";

export const DEFAULT_ACCENT_COLOR = "#4f46e5";

const DEFAULT_FIELD_NAMES = {
    name: "projects.grid_view.name",
    status: "projects.grid_view.status",
    assignee: "projects.grid_view.assignee",
    description: "projects.grid_view.description",
    start_date: "projects.grid_view.start_date",
    due_date: "projects.grid_view.due_date",
};

const SKIPPED_FIELDS = new Set(["id", "created_by", "created_at", "updated_at", "deleted_at"]);

const NEW_FIELD_SORT_OFFSET = 9000;

export function headerLabel(header, translate) {
    const key = DEFAULT_FIELD_NAMES[header?.name];

    return key ? translate(key) : fieldLabel(header);
}

export function emptyFormData(workspaceId, tableId) {
    return normalizeFormData({}, workspaceId, tableId);
}

export function normalizeFormData(data, workspaceId, tableId) {
    return {
        id: data.id || "",
        workspace_id: data.workspace_id || workspaceId,
        table_id: data.table_id || tableId,
        name: data.name || "",
        title: data.title || "",
        description: data.description || "",
        cover_image: data.cover_image || "",
        cover_position: data.cover_position ?? 50,
        logo: data.logo || "",
        accent_color: data.accent_color || DEFAULT_ACCENT_COLOR,
        public_token: data.public_token || "",
        active: data.active ?? true,
        created_by: data.created_by || "",
        updated_at: data.updated_at || 0,
    };
}

// Every field of the table is listed: a saved field keeps its settings, a
// field added to the table since the form was saved comes last and hidden,
// and the name is always shown and required because a row needs one.
export function mergeFormFields(headers, savedFields) {
    const savedByName = new Map(savedFields.map((f) => [f.header_name, f]));

    const merged = headers
        .filter(
            (h) =>
                !SKIPPED_FIELDS.has(h.name) &&
                (h.header_usage || "").toLowerCase() !== "calculations",
        )
        .map((h, i) => {
            const saved = savedByName.get(h.name);

            if (saved) return { ...saved, column_position: saved.column_position || "full" };

            return {
                id: "",
                header_id: h.id || h.name,
                header_name: h.name,
                display_name: fieldLabel(h),
                label: "",
                placeholder: "",
                description: "",
                required: false,
                visible: false,
                sort_order: NEW_FIELD_SORT_OFFSET + i,
                column_position: "full",
                visibility_rule: null,
                field_type: h.header_usage || "",
                linked_id: h.linked_id || "",
                single_select: Boolean(h.single_select),
                parent_table_id: h.parent_table_id || "",
            };
        });

    merged.sort((a, b) => a.sort_order - b.sort_order);

    // A condition on a field since removed from the table would be refused
    // on save.
    const names = new Set(merged.map((f) => f.header_name));

    for (const field of merged) {
        if (field.visibility_rule && !names.has(field.visibility_rule.trigger_field)) {
            field.visibility_rule = null;
        }
    }

    const nameField = merged.find((f) => f.header_name === "name");

    if (nameField) {
        nameField.required = true;
        nameField.visible = true;
    }

    return merged;
}

// Mirrors how SubmitForm reads a value: a link with a single-select target
// takes one option, any other link a list of rows.
export function formFieldType(field) {
    const type = (field.field_type || "").toLowerCase();

    if (field.header_name === "color") return "color";
    if (type === "master link") return "master-link";
    if (field.linked_id) return field.single_select ? "single-select" : "link";
    if (type === "bool" || type === "checkbox") return "checkbox";
    if (type === "date" || type === "default_date") return "date";
    if (type === "assignee" || type === "default_assignee") return "assignee";
    if (["url", "number", "decimal", "file"].includes(type)) return type;

    return "text";
}

// The table whose rows a link field's picker lists, as the server checks it.
export function linkedTableId(field) {
    if (field.single_select) return field.linked_id || "";

    return field.parent_table_id || field.linked_id || "";
}

export function fillableFormFields(fields) {
    return (fields || []).filter((f) => f.visible);
}

export function formImageUrl(workspaceId, fileId) {
    return fileId ? `/api/workspaces/${workspaceId}/file/${fileId}` : "";
}

function isEmptyFormValue(value) {
    return (
        value == null ||
        (typeof value === "string" && value.trim() === "") ||
        value === false ||
        (Array.isArray(value) && value.length === 0)
    );
}

export function formFieldLabel(field, translate) {
    return (
        field.label ||
        headerLabel({ name: field.header_name, display_name: field.display_name }, translate)
    );
}

// Whether text is a whole number, or the start of one, as it is typed: digits
// after an optional leading minus.
export function isWholeNumberText(text) {
    return /^-?\d*$/.test(text);
}

// A number field is a DECIMAL(10,0) column and a decimal field a DECIMAL(20,2).
const NUMBER_LIMITS = { number: 1e10, decimal: 1e18 };

// A row's name is a VARCHAR(255), which counts letters; any other text is a
// TEXT, which counts bytes and so has no letter limit.
const NAME_MAX_LENGTH = 255;
const TEXT_MAX_BYTES = 65535;

export function formFieldMaxLength(field) {
    return field.header_name === "name" ? NAME_MAX_LENGTH : undefined;
}

function fitsTextColumn(field, text) {
    if (field.header_name === "name") return [...text].length <= NAME_MAX_LENGTH;

    return new TextEncoder().encode(text).length <= TEXT_MAX_BYTES;
}

// Answers the server would refuse, with the message saying what each takes.
export function invalidAnswerFields(fields, values) {
    const invalid = {};

    for (const f of fields) {
        const value = values[f.header_name];
        const type = formFieldType(f);

        if (isEmptyFormValue(value)) continue;

        if (type === "number" && !/^\s*-?\d+\s*$/.test(String(value))) {
            invalid[f.header_name] = "projects.form_view.error.whole_number";
        } else if (type === "decimal" && !Number.isFinite(Number(value))) {
            invalid[f.header_name] = "projects.form_view.error.number";
        } else if (NUMBER_LIMITS[type] && Math.abs(Number(value)) >= NUMBER_LIMITS[type]) {
            invalid[f.header_name] = "projects.form_view.error.too_large";
        } else if (typeof value === "string" && !fitsTextColumn(f, value)) {
            invalid[f.header_name] = "projects.form_view.error.too_long";
        }
    }

    return invalid;
}

// A request that got no answer failed on the way, so axios's own "Network
// Error" is the only message it has.
export function formRequestError(error, translate) {
    if (!error?.response) return translate("projects.form_view.error.network");

    return extractErrorMessage(error);
}

export function missingRequiredFields(fields, values) {
    return fields
        .filter((f) => f.required && !f.read_only && isEmptyFormValue(values[f.header_name]))
        .map((f) => f.header_name);
}

// Pickers hold the chosen rows as {id, name}; the server takes their ids.
function submitValue(value) {
    if (Array.isArray(value)) return value.map((item) => item.id).join(",");
    if (value && typeof value === "object") return value.id;

    return value ?? "";
}

export function formSubmitFields(fields, values) {
    return fields.map((f) => ({ name: f.header_name, value: submitValue(values[f.header_name]) }));
}

export function defaultAnswer(field) {
    if (!field.default_value) return undefined;

    switch (formFieldType(field)) {
        case "link":
            return field.default_items || [];
        case "single-select":
            return field.default_items?.[0] ?? null;
        case "checkbox":
            return field.default_value !== "false" && field.default_value !== "0";
        default:
            return field.default_value;
    }
}

// No one can answer such a field, so it stays empty on every submission.
export function readOnlyWithoutDefault(field) {
    return Boolean(field.required && field.read_only && isEmptyFormValue(defaultAnswer(field)));
}

export function defaultAnswers(fields) {
    return Object.fromEntries(
        (fields || []).filter((f) => f.default_value).map((f) => [f.header_name, defaultAnswer(f)]),
    );
}

export function defaultFromAnswer(field, answer) {
    const type = formFieldType(field);

    if (type === "checkbox") return { default_value: answer ? "1" : "", default_items: [] };

    let items = [];

    if (type === "link") items = answer || [];
    if (type === "single-select" && answer) items = [answer];

    return { default_value: String(submitValue(answer)), default_items: items };
}

export const CONDITION_OPERATORS = ["equals", "not_equals", "contains", "is_empty", "is_not_empty"];

export function conditionNeedsValue(operator) {
    return operator !== "is_empty" && operator !== "is_not_empty";
}

const WIDGET_BY_TYPE = { date: "date", assignee: "assignee", checkbox: "checkbox" };

// The input a condition's value is picked with. Contains matches part of
// the text, so it always takes text.
export function conditionValueWidget(field, operator) {
    if (!conditionNeedsValue(operator)) return "none";
    if (operator === "contains") return "text";
    if (field.field_type === "status") return "status";
    if (field.linked_id || field.parent_table_id) return "linked";

    return WIDGET_BY_TYPE[formFieldType(field)] || "text";
}

// The fields whose condition lacks a field to watch or a value to compare,
// which the server would refuse.
export function incompleteConditionFields(fields) {
    return fields
        .filter((f) => {
            const rule = f.visibility_rule;

            if (!rule) return false;

            return !rule.trigger_field || (conditionNeedsValue(rule.operator) && !rule.value);
        })
        .map((f) => f.header_name);
}

// Reads a value as SubmitForm does: a link as the ids of its rows, a checkbox
// as checked or not.
function conditionValues(field, value) {
    const sent = submitValue(value);
    const text = ["string", "number", "boolean"].includes(typeof sent) ? String(sent) : "";

    if (field?.linked_id && !field.single_select) {
        return text
            .split(",")
            .map((id) => id.trim())
            .filter(Boolean);
    }

    if (field && !field.linked_id && ["bool", "checkbox"].includes(field.field_type)) {
        return [String(text !== "" && text !== "false" && text !== "0")];
    }

    return text === "" ? [] : [text];
}

function conditionHolds(rule, values) {
    switch (rule.operator) {
        case "is_empty":
            return values.length === 0;
        case "is_not_empty":
            return values.length > 0;
        case "equals":
            return values.includes(rule.value);
        case "not_equals":
            return !values.includes(rule.value);
        case "contains":
            return values.some((v) => v.includes(rule.value));
        default:
            return false;
    }
}

// The fields whose condition holds. A field that is not shown has no value,
// so a condition watching it sees an empty one.
export function shownFormFields(fields, values) {
    const byName = new Map(fields.map((f) => [f.header_name, f]));

    function shown(field) {
        const rule = field.visibility_rule;

        if (!rule) return true;

        const trigger = byName.get(rule.trigger_field);
        const value = trigger && shown(trigger) ? values[rule.trigger_field] : null;

        return conditionHolds(rule, conditionValues(trigger, value));
    }

    return fields.filter(shown);
}

function watches(fields, field, target) {
    const byName = new Map(fields.map((f) => [f.header_name, f]));
    let current = field;

    for (let i = 0; current?.visibility_rule && i < fields.length; i++) {
        if (current.visibility_rule.trigger_field === target.header_name) return true;
        current = byName.get(current.visibility_rule.trigger_field);
    }

    return false;
}

// The fields a condition on field can watch: the others shown on the form,
// less file fields and those that already depend on field, which would loop.
export function conditionTriggers(fields, field) {
    return fields.filter(
        (f) =>
            f.header_name === field.visibility_rule?.trigger_field ||
            (f.header_name !== field.header_name &&
                f.visible &&
                formFieldType(f) !== "file" &&
                !watches(fields, f, field)),
    );
}
