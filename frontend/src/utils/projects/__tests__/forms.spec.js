// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import { t } from "@/i18n/index.js";
import {
    defaultAnswer,
    defaultAnswers,
    defaultFromAnswer,
    conditionTriggers,
    conditionValueWidget,
    emptyFormData,
    fillableFormFields,
    formFieldType,
    formImageUrl,
    formSubmitFields,
    headerLabel,
    incompleteConditionFields,
    linkedTableId,
    mergeFormFields,
    missingRequiredFields,
    readOnlyWithoutDefault,
    normalizeFormData,
    shownFormFields,
} from "../forms";

describe("headerLabel", () => {
    it("translates a built-in field and reads a custom field's display name", () => {
        expect(headerLabel({ name: "due_date" }, t.value)).toBe("Due Date");
        expect(headerLabel({ name: "budget", display_name: { String: "Budget" } }, t.value)).toBe(
            "Budget",
        );
        expect(headerLabel({ name: "budget", display_name: "Budget" }, t.value)).toBe("Budget");
    });

    it("falls back to the column name", () => {
        expect(headerLabel({ name: "c_123" }, t.value)).toBe("c_123");
        expect(headerLabel(null, t.value)).toBe("");
    });
});

describe("normalizeFormData", () => {
    it("starts a new form active, with the default accent and cover position", () => {
        const form = emptyFormData("w1", "t1");

        expect(form).toMatchObject({
            id: "",
            workspace_id: "w1",
            table_id: "t1",
            active: true,
            accent_color: "#4f46e5",
            cover_position: 50,
        });
    });

    it("keeps what the server stored, including an inactive form", () => {
        const form = normalizeFormData(
            { id: "f1", name: "Intake", active: false, cover_position: 0, accent_color: "#000000" },
            "w1",
            "t1",
        );

        expect(form).toMatchObject({
            id: "f1",
            name: "Intake",
            active: false,
            cover_position: 0,
            accent_color: "#000000",
            workspace_id: "w1",
        });
    });
});

describe("mergeFormFields", () => {
    const headers = [
        { id: "h-id", name: "id" },
        { id: "h-name", name: "name" },
        { id: "h-status", name: "status" },
        { id: "h-total", name: "total", header_usage: "calculations" },
        { id: "h-budget", name: "budget", display_name: { String: "Budget" } },
        { id: "h-created", name: "created_at" },
    ];

    it("lists every editable field, hidden except the name", () => {
        const fields = mergeFormFields(headers, [], t.value);

        expect(fields.map((f) => f.header_name)).toEqual(["name", "status", "budget"]);
        expect(fields.map((f) => f.visible)).toEqual([true, false, false]);
        expect(fields[2]).toMatchObject({
            header_id: "h-budget",
            label: "Budget",
            required: false,
            column_position: "full",
        });
    });

    it("keeps saved fields in their order and adds new ones after them", () => {
        const saved = [
            { header_name: "budget", label: "Cost", visible: true, sort_order: 0 },
            { header_name: "name", label: "Title", visible: true, sort_order: 1 },
        ];

        const fields = mergeFormFields(headers, saved, t.value);

        expect(fields.map((f) => f.header_name)).toEqual(["budget", "name", "status"]);
        expect(fields[0]).toMatchObject({ label: "Cost", column_position: "full" });
    });

    it("reads a new field's type and link from its header", () => {
        const linked = [
            ...headers,
            {
                id: "h-client",
                name: "client",
                header_usage: "link",
                linked_id: "t-link",
                parent_table_id: "t-clients",
            },
        ];

        const client = mergeFormFields(linked, [], t.value).find((f) => f.header_name === "client");

        expect(client).toMatchObject({
            field_type: "link",
            linked_id: "t-link",
            single_select: false,
            parent_table_id: "t-clients",
            visibility_rule: null,
        });
    });

    it("drops a condition on a field the table no longer has", () => {
        const saved = [
            {
                header_name: "budget",
                sort_order: 0,
                visibility_rule: { trigger_field: "gone", operator: "is_empty", value: "" },
            },
            {
                header_name: "status",
                sort_order: 1,
                visibility_rule: { trigger_field: "budget", operator: "is_empty", value: "" },
            },
        ];

        const [budget, status] = mergeFormFields(headers, saved, t.value);

        expect(budget.visibility_rule).toBeNull();
        expect(status.visibility_rule).toEqual(saved[1].visibility_rule);
    });

    it("always shows and requires the name", () => {
        const saved = [{ header_name: "name", visible: false, required: false, sort_order: 0 }];

        const [name] = mergeFormFields(headers, saved, t.value);

        expect(name).toMatchObject({ visible: true, required: true });
    });
});

describe("formFieldType", () => {
    it.each([
        [{ field_type: "text" }, "text"],
        [{ field_type: "" }, "text"],
        [{ field_type: "bool" }, "checkbox"],
        [{ field_type: "default_date", header_name: "due_date" }, "date"],
        [{ field_type: "date" }, "date"],
        [{ field_type: "default_assignee" }, "assignee"],
        [{ field_type: "assignee" }, "assignee"],
        [{ field_type: "master link" }, "master-link"],
        [{ field_type: "status", linked_id: "t-status", single_select: true }, "single-select"],
        [{ field_type: "link", linked_id: "t-other" }, "link"],
        [{ field_type: "number" }, "number"],
        [{ field_type: "decimal" }, "decimal"],
        [{ field_type: "url" }, "url"],
        [{ field_type: "file" }, "file"],
        [{ field_type: "text", header_name: "color" }, "color"],
    ])("reads %o as %s", (field, type) => {
        expect(formFieldType(field)).toBe(type);
    });
});

describe("linkedTableId", () => {
    it("lists a link's parent table when it has one", () => {
        expect(linkedTableId({ linked_id: "t-child", parent_table_id: "t-parent" })).toBe(
            "t-parent",
        );
        expect(linkedTableId({ linked_id: "t-status" })).toBe("t-status");
        expect(linkedTableId({})).toBe("");
    });

    it("lists a single-select's own options, not the table it belongs to", () => {
        expect(
            linkedTableId({
                linked_id: "t-status",
                single_select: true,
                parent_table_id: "t-form",
            }),
        ).toBe("t-status");
    });
});

describe("fillableFormFields", () => {
    it("keeps the visible fields in order, file fields among them", () => {
        const fields = [
            { header_name: "name", visible: true },
            { header_name: "hidden", visible: false },
            { header_name: "files", visible: true, field_type: "file" },
            { header_name: "notes", visible: true },
        ];

        expect(fillableFormFields(fields).map((f) => f.header_name)).toEqual([
            "name",
            "files",
            "notes",
        ]);
        expect(fillableFormFields(undefined)).toEqual([]);
    });
});

describe("formImageUrl", () => {
    it("reads a form's image through the workspace's file route", () => {
        expect(formImageUrl("w1", "img1")).toBe("/api/workspaces/w1/file/img1");
        expect(formImageUrl("w1", "")).toBe("");
    });
});

describe("missingRequiredFields", () => {
    it("names required fields left empty, an unticked box among them", () => {
        const fields = [
            { header_name: "name", required: true },
            { header_name: "tags", required: true },
            { header_name: "agree", required: true },
            { header_name: "notes", required: false },
        ];

        expect(missingRequiredFields(fields, { name: "", tags: [], agree: false })).toEqual([
            "name",
            "tags",
            "agree",
        ]);
        expect(missingRequiredFields(fields, { name: "A", tags: [{ id: "1" }] })).toEqual([
            "agree",
        ]);
        expect(
            missingRequiredFields(fields, { name: "A", tags: [{ id: "1" }], agree: true }),
        ).toEqual([]);
    });

    it("leaves out a read-only field, which the server fills from its default", () => {
        const fields = [{ header_name: "code", required: true, read_only: true }];

        expect(missingRequiredFields(fields, {})).toEqual([]);
    });

    it("counts an answer of only spaces as empty", () => {
        const fields = [{ header_name: "name", required: true }];

        expect(missingRequiredFields(fields, { name: "   " })).toEqual(["name"]);
    });
});

describe("formSubmitFields", () => {
    it("sends picked rows by id and an empty field as an empty string", () => {
        const fields = ["name", "status", "tags", "done", "notes", "docs"].map((header_name) => ({
            header_name,
        }));
        const values = {
            name: "Task",
            status: { id: "s1", name: "Open" },
            tags: [
                { id: "r1", name: "One" },
                { id: "r2", name: "Two" },
            ],
            done: false,
            docs: [
                { id: "file1", name: "a.pdf" },
                { id: "file2", name: "b.png" },
            ],
        };

        expect(formSubmitFields(fields, values)).toEqual([
            { name: "name", value: "Task" },
            { name: "status", value: "s1" },
            { name: "tags", value: "r1,r2" },
            { name: "done", value: false },
            { name: "notes", value: "" },
            { name: "docs", value: "file1,file2" },
        ]);
    });
});

describe("shownFormFields", () => {
    const rule = (trigger_field, operator, value = "") => ({ trigger_field, operator, value });
    const names = (fields, values) => shownFormFields(fields, values).map((f) => f.header_name);

    it.each([
        ["equals", "refund", "refund", true],
        ["equals", "refund", "question", false],
        ["not_equals", "refund", "question", true],
        ["not_equals", "refund", "", true],
        ["contains", "fun", "refund", true],
        ["contains", "fun", "question", false],
        ["is_empty", "", "", true],
        ["is_empty", "", undefined, true],
        ["is_not_empty", "", "x", true],
        ["is_not_empty", "", "", false],
    ])("%s %j against %j shows the field: %s", (operator, value, kind, shown) => {
        const fields = [
            { header_name: "kind" },
            { header_name: "reason", visibility_rule: rule("kind", operator, value) },
        ];

        expect(names(fields, { kind }).includes("reason")).toBe(shown);
    });

    it("compares picked rows by id", () => {
        const fields = [
            { header_name: "clients", linked_id: "t-link" },
            { header_name: "status", linked_id: "t-status", single_select: true },
            { header_name: "a", visibility_rule: rule("clients", "equals", "r2") },
            { header_name: "b", visibility_rule: rule("status", "equals", "s1") },
            { header_name: "c", visibility_rule: rule("clients", "not_equals", "r3") },
        ];
        const values = {
            clients: [
                { id: "r1", name: "One" },
                { id: "r2", name: "Two" },
            ],
            status: { id: "s1", name: "Open" },
        };

        expect(names(fields, values)).toEqual(["clients", "status", "a", "b", "c"]);
        expect(names(fields, {})).toEqual(["clients", "status", "c"]);
    });

    it("reads an untouched checkbox as unchecked", () => {
        const fields = [
            { header_name: "urgent", field_type: "bool" },
            { header_name: "calm", visibility_rule: rule("urgent", "equals", "false") },
            { header_name: "why", visibility_rule: rule("urgent", "equals", "true") },
        ];

        expect(names(fields, {})).toEqual(["urgent", "calm"]);
        expect(names(fields, { urgent: true })).toEqual(["urgent", "why"]);
    });

    it("sees a hidden field as empty", () => {
        const fields = [
            { header_name: "kind" },
            { header_name: "reason", visibility_rule: rule("kind", "equals", "refund") },
            { header_name: "other", visibility_rule: rule("reason", "is_empty") },
            { header_name: "gone", visibility_rule: rule("missing", "is_not_empty") },
        ];

        expect(names(fields, { kind: "question", reason: "kept from before" })).toEqual([
            "kind",
            "other",
        ]);
    });
});

describe("conditionTriggers", () => {
    it("offers the other shown fields, less files and those watching the field", () => {
        const name = { header_name: "name", visible: true };
        const hidden = { header_name: "hidden", visible: false };
        const files = { header_name: "files", visible: true, field_type: "file" };
        const notes = { header_name: "notes", visible: true };
        const reason = {
            header_name: "reason",
            visible: true,
            visibility_rule: { trigger_field: "name", operator: "is_empty", value: "" },
        };
        const fields = [name, hidden, files, notes, reason];

        expect(conditionTriggers(fields, name).map((f) => f.header_name)).toEqual(["notes"]);
        expect(conditionTriggers(fields, notes).map((f) => f.header_name)).toEqual([
            "name",
            "reason",
        ]);
    });
});

describe("conditionValueWidget", () => {
    it.each([
        [{ field_type: "status", linked_id: "t-status", single_select: true }, "status"],
        [{ field_type: "link", linked_id: "t-link", parent_table_id: "t-clients" }, "linked"],
        [{ field_type: "single select", linked_id: "t-options", single_select: true }, "linked"],
        [{ field_type: "", linked_id: "t-link" }, "linked"],
        [{ field_type: "default_assignee" }, "assignee"],
        [{ field_type: "date" }, "date"],
        [{ field_type: "default_date" }, "date"],
        [{ field_type: "bool" }, "checkbox"],
        [{ field_type: "number" }, "text"],
        [{ field_type: "text" }, "text"],
    ])("picks a value for %o with %s", (field, widget) => {
        expect(conditionValueWidget(field, "equals")).toBe(widget);
        expect(conditionValueWidget(field, "not_equals")).toBe(widget);
    });

    it("takes text for contains and nothing for the emptiness checks", () => {
        const field = { field_type: "link", linked_id: "t-link" };

        expect(conditionValueWidget(field, "contains")).toBe("text");
        expect(conditionValueWidget(field, "is_empty")).toBe("none");
        expect(conditionValueWidget(field, "is_not_empty")).toBe("none");
    });
});

describe("incompleteConditionFields", () => {
    it("names the fields whose condition lacks a field or a value", () => {
        const rule = (trigger_field, operator, value) => ({ trigger_field, operator, value });
        const fields = [
            { header_name: "plain", visibility_rule: null },
            { header_name: "no_field", visibility_rule: rule("", "equals", "x") },
            { header_name: "no_value", visibility_rule: rule("name", "equals", "") },
            { header_name: "empty_check", visibility_rule: rule("name", "is_empty", "") },
            { header_name: "done", visibility_rule: rule("name", "equals", "x") },
        ];

        expect(incompleteConditionFields(fields)).toEqual(["no_field", "no_value"]);
    });
});

describe("form field defaults", () => {
    const acme = { id: "r1", name: "Acme" };
    const open = { id: "o1", name: "Open", color: "#16a34a" };

    it.each([
        ["text", { field_type: "text", default_value: "Hello" }, "Hello"],
        ["number", { field_type: "number", default_value: "12.5" }, "12.5"],
        ["date", { field_type: "date", default_value: "2026-11-05" }, "2026-11-05"],
        ["ticked checkbox", { field_type: "bool", default_value: "1" }, true],
        ["person", { field_type: "assignee", default_value: "u1" }, "u1"],
        ["link", { linked_id: "t2", default_value: "r1", default_items: [acme] }, [acme]],
        [
            "option",
            { linked_id: "t-s", single_select: true, default_value: "o1", default_items: [open] },
            open,
        ],
    ])("starts a %s field with its default", (_, field, want) => {
        expect(defaultAnswer(field)).toEqual(want);
    });

    it("starts a field without a default empty", () => {
        expect(defaultAnswer({ field_type: "text", default_value: "" })).toBeUndefined();
        expect(defaultAnswers([{ header_name: "a", field_type: "text" }])).toEqual({});
    });

    it.each([
        ["text", { field_type: "text" }, "Hello", { default_value: "Hello", default_items: [] }],
        ["checkbox", { field_type: "bool" }, true, { default_value: "1", default_items: [] }],
        [
            "cleared checkbox",
            { field_type: "bool" },
            false,
            { default_value: "", default_items: [] },
        ],
        ["link", { linked_id: "t2" }, [acme], { default_value: "r1", default_items: [acme] }],
        [
            "option",
            { linked_id: "t-s", single_select: true },
            open,
            { default_value: "o1", default_items: [open] },
        ],
        [
            "cleared option",
            { linked_id: "t-s", single_select: true },
            null,
            { default_value: "", default_items: [] },
        ],
    ])("keeps a %s default as its answer would be sent", (_, field, answer, want) => {
        expect(defaultFromAnswer(field, answer)).toEqual(want);
    });
});

describe("readOnlyWithoutDefault", () => {
    it("flags a required read-only field with no default", () => {
        expect(readOnlyWithoutDefault({ required: true, read_only: true, default_value: "" })).toBe(
            true,
        );
    });

    it("does not flag one with a default, or one that is not both", () => {
        const field = { field_type: "text", required: true, read_only: true, default_value: "A" };

        expect(readOnlyWithoutDefault(field)).toBe(false);
        expect(readOnlyWithoutDefault({ ...field, default_value: "", read_only: false })).toBe(
            false,
        );
        expect(readOnlyWithoutDefault({ ...field, default_value: "", required: false })).toBe(
            false,
        );
    });
});
