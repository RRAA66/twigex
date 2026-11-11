// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormFieldInput from "@/components/Projects/Form/FormFieldInput.vue";

vi.mock("@/services/formService", () => ({
    default: { getPublicLinkedItems: vi.fn() },
}));

let w;

function mountField(field, props = {}) {
    setActivePinia(createPinia());
    w = mount(FormFieldInput, {
        props: { field: { header_name: "f", label: "Agree", ...field }, token: "tok", ...props },
        attachTo: document.body,
    });

    return w;
}

function labelFor(control) {
    return document.querySelector(`label[for='${control.id}']`)?.textContent.trim() ?? "";
}

describe("FormFieldInput", () => {
    afterEach(() => {
        w?.unmount();
    });

    it.each([
        ["text", "input"],
        ["checkbox", "input[type='checkbox']"],
        ["number", "input[type='number']"],
    ])("labels a %s field with its own label", (fieldType, selector) => {
        mountField({ field_type: fieldType });

        expect(labelFor(w.find(selector).element)).toContain("Agree");
    });

    it("labels a colour field with its own label", () => {
        mountField({ header_name: "color" });

        expect(labelFor(w.find("input[type='color']").element)).toContain("Agree");
    });

    it.each([
        ["link", { linked_id: "t-x" }],
        ["single-select", { linked_id: "t-x", single_select: true }],
        ["person", { field_type: "assignee" }],
    ])("names a %s field's search box by its label", (_, field) => {
        mountField(field);

        const labelledBy = w.find("input[role='combobox']").attributes("aria-labelledby");

        expect(document.getElementById(labelledBy)?.textContent).toContain("Agree");
    });

    it("labels a date field's button", () => {
        mountField({ field_type: "date" });

        expect(labelFor(w.find("button").element)).toContain("Agree");
    });

    it("labels a master link field's list button and marks it invalid", async () => {
        mountField({ field_type: "master link", required: true }, { error: "Required" });
        await nextTick();

        const button = w.find("button");
        const labels = button
            .attributes("aria-labelledby")
            .split(" ")
            .map((id) => document.getElementById(id)?.textContent.trim());

        expect(labels).toContain("Agree *");
        expect(button.attributes("aria-invalid")).toBe("true");
        expect(w.text()).toContain("Required");
    });

    it.each([
        ["text", { field_type: "text" }, "input"],
        ["checkbox", { field_type: "checkbox" }, "input"],
        ["link", { linked_id: "t-x" }, "input[role='combobox']"],
    ])("ties a %s field's error and description to it", (_, field, selector) => {
        mountField({ ...field, required: true, description: "Why" }, { error: "Required" });

        const input = w.find(selector);
        const describedBy = input.attributes("aria-describedby").split(" ");

        expect(describedBy.map((id) => document.getElementById(id).textContent.trim())).toEqual([
            "Why",
            "Required",
        ]);
        expect(input.attributes("aria-invalid")).toBe("true");
        expect(input.attributes("aria-required")).toBe("true");
    });

    it("leaves a valid optional field unmarked", () => {
        mountField({ field_type: "text" });

        const input = w.find("input");

        expect(input.attributes("aria-describedby")).toBeUndefined();
        expect(input.attributes("aria-invalid")).toBeUndefined();
        expect(input.attributes("aria-required")).toBeUndefined();
    });

    it("shows the error on a date field", () => {
        mountField({ field_type: "date" }, { error: "Required" });

        expect(w.find("button").attributes("aria-invalid")).toBe("true");
        expect(w.text()).toContain("Required");
    });

    it("shows an unset colour as not chosen and sends no colour", () => {
        mountField({ header_name: "color" });

        expect(w.text()).toContain("No color chosen");
        expect(w.emitted("update:modelValue")).toBeUndefined();
    });

    it("clears a chosen colour", async () => {
        mountField({ header_name: "color" }, { modelValue: "#ff0000" });

        await w.find("button").trigger("click");

        expect(w.emitted("update:modelValue")).toEqual([[""]]);
    });
});
