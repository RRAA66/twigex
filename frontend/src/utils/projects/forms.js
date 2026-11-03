// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { t } from "@/i18n/index.js";
import { fieldLabel } from "@/utils/projects/rows";

const DEFAULT_ACCENT_COLOR = "#4f46e5";

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

export function headerLabel(header) {
    const key = DEFAULT_FIELD_NAMES[header?.name];

    return key ? t.value(key) : fieldLabel(header);
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
        requires_login: data.requires_login || false,
        active: data.active ?? true,
        created_by: data.created_by || "",
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
                label: headerLabel(h),
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

// The input a form field is filled with. Mirrors how SubmitForm reads the
// value: a link with a single-select target takes one option, any other link
// a list of rows.
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
    return field.parent_table_id || field.linked_id || "";
}

// File fields come with uploads, which the form cannot send yet.
export function fillableFormFields(fields) {
    return (fields || []).filter((f) => f.visible && formFieldType(f) !== "file");
}

function isEmptyFormValue(value) {
    return value == null || value === "" || (Array.isArray(value) && value.length === 0);
}

export function missingRequiredFields(fields, values) {
    return fields
        .filter((f) => f.required && isEmptyFormValue(values[f.header_name]))
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

export const CONDITION_OPERATORS = ["equals", "not_equals", "contains", "is_empty", "is_not_empty"];

export function conditionNeedsValue(operator) {
    return operator !== "is_empty" && operator !== "is_not_empty";
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
