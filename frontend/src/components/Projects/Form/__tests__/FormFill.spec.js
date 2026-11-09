// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormFill from "@/components/Projects/Form/FormFill.vue";
import formService from "@/services/formService";

vi.mock("@/js/websocket", () => ({
    initialize: vi.fn(),
    close: vi.fn(),
    onProjectChange: () => () => {},
}));

vi.mock("@/services/formService", () => ({
    default: { getPublicForm: vi.fn(), submitPublicForm: vi.fn(), uploadPublicFormFiles: vi.fn() },
}));

const form = {
    id: "f1",
    title: "Intake",
    description: "Tell us",
    accent_color: "#ff0000",
    can_submit: true,
    workspace_id: "w1",
    link_tables: [],
    fields: [
        {
            header_name: "name",
            label: "Title",
            field_type: "text",
            visible: true,
            required: true,
            column_position: "left",
        },
        { header_name: "budget", label: "Budget", field_type: "number", visible: true },
        { header_name: "done", label: "Done", field_type: "bool", visible: true },
        { header_name: "due_date", label: "Due", field_type: "default_date", visible: true },
        { header_name: "assignee", label: "Owner", field_type: "default_assignee", visible: true },
        { header_name: "files", label: "Files", field_type: "file", visible: true },
    ],
};

async function mountFill() {
    const pinia = createPinia();

    setActivePinia(pinia);
    const w = mount(FormFill, {
        props: { token: "tok" },
        global: { plugins: [pinia] },
        attachTo: document.body,
    });

    await flushPromises();

    return w;
}

describe("FormFill", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        formService.getPublicForm.mockResolvedValue({ data: form });
    });

    it("shows the form's fields in order, with a hidden honeypot", async () => {
        const w = await mountFill();

        expect(w.text()).toContain("Intake");
        expect(w.text()).toContain("Tell us");
        expect(w.find("form").attributes("style")).toContain("border-top-color");

        const labels = w.findAll("label.mb-1").map((l) => l.text().replace(/\s+/g, " "));

        expect(labels).toEqual(["Title *", "Budget", "Done", "Due", "Owner", "Files"]);
        expect(w.find("input[type='number']").exists()).toBe(true);
        expect(w.find("input[type='checkbox']").exists()).toBe(true);
        expect(w.text()).toContain("Pick a date");
        expect(w.find("input[placeholder='Select person']").exists()).toBe(true);

        const honeypot = w.find("input[tabindex='-1']");

        expect(honeypot.attributes("autocomplete")).toBe("off");
        expect(honeypot.element.parentElement.getAttribute("aria-hidden")).toBe("true");
        w.unmount();
    });

    it("marks an empty required field instead of sending", async () => {
        const w = await mountFill();

        await w.find("form").trigger("submit");
        await flushPromises();

        expect(formService.submitPublicForm).not.toHaveBeenCalled();
        expect(w.text()).toContain("This field is required");
        w.unmount();
    });

    it("thanks the user and lets them answer again", async () => {
        formService.submitPublicForm.mockResolvedValue({ data: {} });
        const w = await mountFill();

        await w.find("input[type='text']:not([tabindex])").setValue("Task");
        await w.find("form").trigger("submit");
        await flushPromises();

        expect(w.text()).toContain("Thank you!");

        await w.find("button").trigger("click");

        expect(w.find("form").exists()).toBe(true);
        w.unmount();
    });

    it("shows why the server refused the answer", async () => {
        formService.submitPublicForm.mockRejectedValue({
            response: { status: 400, data: { error: "The due date is before the start date." } },
        });
        const w = await mountFill();

        await w.find("input[type='text']:not([tabindex])").setValue("Task");
        await w.find("form").trigger("submit");
        await flushPromises();

        expect(w.find("[role='alert']").text()).toBe("The due date is before the start date.");
        w.unmount();
    });

    it("uploads files for a file field and sends their ids", async () => {
        formService.uploadPublicFormFiles.mockResolvedValue({
            data: [{ id: "file1", name: "report.pdf", size: 3, type: "application/pdf" }],
        });
        formService.submitPublicForm.mockResolvedValue({ data: {} });
        const w = await mountFill();
        const picked = new File(["pdf"], "report.pdf", { type: "application/pdf" });
        const input = w.find("input[type='file']");

        Object.defineProperty(input.element, "files", { value: [picked] });
        await input.trigger("change");
        await flushPromises();

        expect(formService.uploadPublicFormFiles).toHaveBeenCalledWith("tok", [picked]);
        expect(w.text()).toContain("report.pdf");

        await w.find("input[type='text']:not([tabindex])").setValue("Task");
        await w.find("form").trigger("submit");
        await flushPromises();

        const [, sent] = formService.submitPublicForm.mock.calls[0];

        expect(sent).toContainEqual({ name: "files", value: "file1" });
        w.unmount();
    });

    it("shows the form without a way to send it to someone who may not add rows", async () => {
        formService.getPublicForm.mockResolvedValue({ data: { ...form, can_submit: false } });
        const w = await mountFill();

        expect(w.text()).toContain(
            "You can view this form but you're not allowed to add records to this table.",
        );
        expect(w.find("fieldset").attributes("disabled")).toBeDefined();
        expect(w.find("button[type='submit']").exists()).toBe(false);
        w.unmount();
    });

    it.each([
        [403, "You don't have access to this form."],
        [404, "This form doesn't exist or is no longer open."],
    ])("explains a %i", async (status, message) => {
        formService.getPublicForm.mockRejectedValue({ response: { status } });

        const w = await mountFill();

        expect(w.text()).toContain(message);
        expect(w.find("form").exists()).toBe(false);
        w.unmount();
    });
});
