// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import { t } from "@/i18n/index.js";
import {
    emptyFormData,
    fillableFormFields,
    formFieldType,
    formSubmitFields,
    headerLabel,
    linkedTableId,
    mergeFormFields,
    missingRequiredFields,
    normalizeFormData,
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

    it("keeps the version the server stored, so a save can name it", () => {
        expect(normalizeFormData({ id: "f1", updated_at: 1700 }, "w1", "t1").updated_at).toBe(1700);
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
        const fields = mergeFormFields(headers, []);

        expect(fields.map((f) => f.header_name)).toEqual(["name", "status", "budget"]);
        expect(fields.map((f) => f.visible)).toEqual([true, false, false]);
        expect(fields[2]).toMatchObject({
            header_id: "h-budget",
            label: "",
            required: false,
            column_position: "full",
        });
    });

    it("keeps saved fields in their order and adds new ones after them", () => {
        const saved = [
            { header_name: "budget", label: "Cost", visible: true, sort_order: 0 },
            { header_name: "name", label: "Title", visible: true, sort_order: 1 },
        ];

        const fields = mergeFormFields(headers, saved);

        expect(fields.map((f) => f.header_name)).toEqual(["budget", "name", "status"]);
        expect(fields[0]).toMatchObject({ label: "Cost", column_position: "full" });
    });

    it("always shows and requires the name", () => {
        const saved = [{ header_name: "name", visible: false, required: false, sort_order: 0 }];

        const [name] = mergeFormFields(headers, saved);

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
    it("keeps the visible fields in order, without file fields", () => {
        const fields = [
            { header_name: "name", visible: true },
            { header_name: "hidden", visible: false },
            { header_name: "files", visible: true, field_type: "file" },
            { header_name: "notes", visible: true },
        ];

        expect(fillableFormFields(fields).map((f) => f.header_name)).toEqual(["name", "notes"]);
        expect(fillableFormFields(undefined)).toEqual([]);
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

    it("counts an answer of only spaces as empty", () => {
        const fields = [{ header_name: "name", required: true }];

        expect(missingRequiredFields(fields, { name: "   " })).toEqual(["name"]);
    });
});

describe("formSubmitFields", () => {
    it("sends picked rows by id and an empty field as an empty string", () => {
        const fields = ["name", "status", "tags", "done", "notes"].map((header_name) => ({
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
        };

        expect(formSubmitFields(fields, values)).toEqual([
            { name: "name", value: "Task" },
            { name: "status", value: "s1" },
            { name: "tags", value: "r1,r2" },
            { name: "done", value: false },
            { name: "notes", value: "" },
        ]);
    });
});
