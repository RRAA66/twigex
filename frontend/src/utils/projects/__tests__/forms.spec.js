// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import { emptyFormData, headerLabel, mergeFormFields, normalizeFormData } from "../forms";

describe("headerLabel", () => {
    it("translates a built-in field and reads a custom field's display name", () => {
        expect(headerLabel({ name: "due_date" })).toBe("Due Date");
        expect(headerLabel({ name: "budget", display_name: { String: "Budget" } })).toBe("Budget");
        expect(headerLabel({ name: "budget", display_name: "Budget" })).toBe("Budget");
    });

    it("falls back to the column name", () => {
        expect(headerLabel({ name: "c_123" })).toBe("c_123");
        expect(headerLabel(null)).toBe("");
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
        const fields = mergeFormFields(headers, []);

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
