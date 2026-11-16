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
        ["decimal", "input[type='number']"],
        ["number", "input[inputmode='numeric']"],
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

    it("shows a read-only field's default instead of an input", () => {
        mountField({
            linked_id: "t-s",
            single_select: true,
            read_only: true,
            default_value: "o1",
            default_items: [{ id: "o1", name: "Open", color: "#16a34a" }],
        });

        const box = w.find("[aria-readonly='true']");

        expect(box.text()).toBe("Open");
        expect(document.getElementById(box.attributes("aria-labelledby")).textContent).toContain(
            "Agree",
        );
        expect(w.find("input").exists()).toBe(false);
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

    it("labels a file field's button and ties its error and description to it", () => {
        setActivePinia(createPinia());
        mountField(
            { field_type: "file", required: true, description: "Why" },
            { error: "Required" },
        );

        const button = w.find("button");
        const describedBy = button.attributes("aria-describedby").split(" ");

        expect(labelFor(button.element)).toContain("Agree");
        expect(describedBy.map((id) => document.getElementById(id).textContent.trim())).toEqual([
            "Why",
            "Required",
        ]);
        expect(button.attributes("aria-invalid")).toBe("true");
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

    it("keeps a whole number field to digits and refuses a decimal separator", async () => {
        mountField({ field_type: "number" }, { modelValue: "12" });

        const input = w.find("input");

        await input.setValue("12.");
        await input.setValue("1,5");
        await input.setValue(" 1 200x");

        expect(w.emitted("update:modelValue")).toEqual([["12"], ["12"], ["1200"]]);
        expect(input.element.value).toBe("1200");
    });

    it("names a field with no label of its own after its table field, translated", () => {
        mountField({ header_name: "due_date", label: "", field_type: "default_date" });

        expect(labelFor(w.find("button").element)).toContain("Due Date");

        w.unmount();
        mountField({
            header_name: "budget",
            label: "",
            display_name: "Cost",
            field_type: "decimal",
        });

        expect(labelFor(w.find("input").element)).toContain("Cost");
    });

    it("clears a chosen colour", async () => {
        mountField({ header_name: "color" }, { modelValue: "#ff0000" });

        await w.find("button").trigger("click");

        expect(w.emitted("update:modelValue")).toEqual([[""]]);
    });
});
