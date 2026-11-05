// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
    default: { getPublicForm: vi.fn(), submitPublicForm: vi.fn() },
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

async function mountFill(props = {}) {
    const pinia = createPinia();

    setActivePinia(pinia);
    const w = mount(FormFill, {
        props: { token: "tok", ...props },
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

        expect(labels).toEqual(["Title *", "Budget", "Done", "Due", "Owner"]);
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

    describe("after it is sent", () => {
        async function send(extra, props) {
            formService.getPublicForm.mockResolvedValue({ data: { ...form, ...extra } });
            formService.submitPublicForm.mockResolvedValue({ data: {} });
            const w = await mountFill(props);

            await w.find("input[type='text']:not([tabindex])").setValue("Task");
            await w.find("form").trigger("submit");
            await flushPromises();

            return w;
        }

        afterEach(() => vi.unstubAllGlobals());

        it("thanks the user in the form's own words", async () => {
            const w = await send({ success_message: "We will call you." });

            expect(w.text()).toContain("We will call you.");
            expect(w.text()).not.toContain("Your response has been recorded.");
            w.unmount();
        });

        it("opens the form's next page from its link", async () => {
            const assign = vi.fn();

            vi.stubGlobal("location", { ...window.location, assign });
            const w = await send(
                { redirect_url: "https://example.com/next" },
                { followRedirect: true },
            );

            expect(assign).toHaveBeenCalledWith("https://example.com/next");
            expect(w.find("a[href='https://example.com/next']").exists()).toBe(true);
            w.unmount();
        });

        it("stays on the page in the builder's preview", async () => {
            const assign = vi.fn();

            vi.stubGlobal("location", { ...window.location, assign });
            const w = await send({ redirect_url: "https://example.com/next" });

            expect(assign).not.toHaveBeenCalled();
            expect(w.text()).toContain("Submit another response");
            w.unmount();
        });
    });

    it("shows and requires a field only while its condition holds", async () => {
        formService.getPublicForm.mockResolvedValue({
            data: {
                ...form,
                fields: [
                    form.fields[0],
                    form.fields[2],
                    {
                        header_name: "reason",
                        label: "Reason",
                        field_type: "text",
                        visible: true,
                        required: true,
                        visibility_rule: {
                            trigger_field: "done",
                            operator: "equals",
                            value: "true",
                        },
                    },
                ],
            },
        });
        formService.submitPublicForm.mockResolvedValue({ data: {} });
        const w = await mountFill();

        expect(w.text()).not.toContain("Reason");

        await w.find("input[type='checkbox']").setValue(true);

        expect(w.text()).toContain("Reason");

        await w.find("input[type='checkbox']").setValue(false);
        await w.find("input[type='text']:not([tabindex])").setValue("Task");
        await w.find("form").trigger("submit");
        await flushPromises();

        expect(formService.submitPublicForm).toHaveBeenCalledWith(
            "tok",
            [
                { name: "name", value: "Task" },
                { name: "done", value: false },
            ],
            "",
        );
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
