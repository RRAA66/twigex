// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormViewFieldRow from "@/components/Projects/Form/FormViewFieldRow.vue";

vi.mock("@/services/workspaceService", () => ({
    default: { getLinkedRecordsLite: vi.fn(), getTableStatusTypes: vi.fn() },
}));

let w;

async function mountRow(field) {
    w = mount(FormViewFieldRow, {
        props: {
            field: { header_name: "budget", label: "Budget", visible: true, ...field },
            fields: [],
            workspaceId: "w1",
            tableId: "t1",
        },
        attachTo: document.body,
    });
    await w.find("button[aria-expanded]").trigger("click");

    return w;
}

function inputLabelled(text) {
    const label = w.findAll("label").find((l) => l.text().startsWith(text));

    return label && w.find(`[id="${label.attributes("for")}"]`);
}

describe("FormViewFieldRow", () => {
    beforeEach(() => setActivePinia(createPinia()));

    afterEach(() => w?.unmount());

    it("keeps a field's default as it would be answered", async () => {
        await mountRow({ field_type: "text" });

        await inputLabelled("Default value").setValue("Website");

        expect(w.emitted("update").at(-1)).toEqual([
            { default_value: "Website", default_items: [] },
        ]);
    });

    it("makes a field read-only", async () => {
        await mountRow({ field_type: "text", default_value: "Website" });

        await inputLabelled("Read-only").setValue(true);

        expect(w.emitted("update").at(-1)).toEqual([{ read_only: true }]);
    });

    it.each([
        ["file", { field_type: "file" }],
        ["master link", { field_type: "master link" }],
    ])("offers no default for a %s field", async (_, field) => {
        await mountRow(field);

        expect(w.text()).not.toContain("Default value");
        expect(w.text()).not.toContain("Read-only");
    });
});
