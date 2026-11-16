// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter } from "vue-router";
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

    it("leaves an answer hidden by a condition out of a calculation", async () => {
        formService.getPublicForm.mockResolvedValue({
            data: {
                ...form,
                fields: [
                    { header_name: "kind", label: "Kind", field_type: "text", visible: true },
                    {
                        header_name: "extra",
                        label: "Extra",
                        field_type: "number",
                        visible: true,
                        visibility_rule: { trigger_field: "kind", operator: "is_empty", value: "" },
                    },
                    {
                        header_name: "total",
                        label: "Total",
                        field_type: "calculations",
                        visible: true,
                        formula: { name: "SUM", args: [{ type: "field", value: "extra" }] },
                    },
                ],
            },
        });
        const w = await mountFill();
        const total = () => w.find("[aria-readonly='true']").text();

        await w.find("input[inputmode='numeric']").setValue("5");

        expect(total()).toBe("5");

        await w.find("fieldset input[type='text']").setValue("paid");

        expect(total()).not.toBe("5");
    });

    it("shows the form's fields in order, with a hidden honeypot", async () => {
        const w = await mountFill();

        expect(w.text()).toContain("Intake");
        expect(w.text()).toContain("Tell us");
        expect(w.find("form").attributes("style")).toContain("border-top-color");

        const labels = w.findAll("label.mb-1").map((l) => l.text().replace(/\s+/g, " "));

        expect(labels).toEqual(["Title *", "Budget", "Done", "Due", "Owner", "Files"]);
        expect(w.find("input[inputmode='numeric']").exists()).toBe(true);
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

    it("moves to the first field in error after a refused send", async () => {
        const w = await mountFill();

        await w.find("form").trigger("submit");
        await flushPromises();

        expect(document.activeElement).toBe(w.find("input[aria-invalid='true']").element);
        expect(document.activeElement.id).toBe(w.findAll("label.mb-1")[0].attributes("for"));
        w.unmount();
    });

    it("moves to the server's reason when it refused the answer", async () => {
        formService.submitPublicForm.mockRejectedValue(new Error("Network Error"));
        const w = await mountFill();

        await w.find("input[type='text']:not([tabindex])").setValue("Task");
        await w.find("form").trigger("submit");
        await flushPromises();

        const alert = w.find("[role='alert']");

        expect(alert.text()).toBe(
            "Could not reach the server. Check your connection and try again.",
        );
        expect(document.activeElement).toBe(alert.element);
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

        function pressContinue(w) {
            return w
                .findAll("button")
                .find((b) => b.text() === "Continue")
                .trigger("click");
        }

        it("opens the form's link only when the user goes on", async () => {
            const assign = vi.fn();

            vi.stubGlobal("location", { ...window.location, assign });
            const w = await send(
                {
                    after_submit: "link",
                    redirect_url: "https://example.com/next",
                    success_message: "We will call you.",
                },
                { followRedirect: true },
            );

            expect(w.text()).toContain("We will call you.");
            expect(assign).not.toHaveBeenCalled();
            await pressContinue(w);
            expect(assign).toHaveBeenCalledWith("https://example.com/next");
            w.unmount();
        });

        it("opens the form's table from its link", async () => {
            const router = createRouter({
                history: createMemoryHistory(),
                routes: [
                    { path: "/", component: { template: "<div/>" } },
                    {
                        path: "/projects/:id/grid/:tid/view/:fid?",
                        name: "grid-view",
                        component: { template: "<div/>" },
                    },
                ],
            });
            const push = vi.spyOn(router, "push");

            formService.getPublicForm.mockResolvedValue({
                data: {
                    ...form,
                    workspace_id: "w1",
                    table_id: "t1",
                    after_submit: "grid",
                    grid_view_id: "v1",
                },
            });
            formService.submitPublicForm.mockResolvedValue({ data: {} });
            setActivePinia(createPinia());
            const w = mount(FormFill, {
                props: { token: "tok", followRedirect: true },
                global: { plugins: [router] },
                attachTo: document.body,
            });

            await flushPromises();
            await w.find("input[type='text']:not([tabindex])").setValue("Task");
            await w.find("form").trigger("submit");
            await flushPromises();
            expect(push).not.toHaveBeenCalled();
            await pressContinue(w);

            expect(push).toHaveBeenCalledWith("/projects/w1/grid/t1/view/v1");
            w.unmount();
        });

        it("stays on the page in the builder's preview", async () => {
            const assign = vi.fn();

            vi.stubGlobal("location", { ...window.location, assign });
            const w = await send({
                after_submit: "link",
                redirect_url: "https://example.com/next",
            });

            expect(assign).not.toHaveBeenCalled();
            expect(w.text()).toContain("Submit another response");
            w.unmount();
        });

        it("sends nothing from the builder's preview", async () => {
            const w = await send({}, { preview: true });

            expect(formService.submitPublicForm).not.toHaveBeenCalled();
            expect(w.text()).toContain("Thank you");
            expect(w.text()).toContain("nothing sent here is saved");
            w.unmount();
        });

        it("still asks for required answers in the preview", async () => {
            formService.getPublicForm.mockResolvedValue({ data: form });
            const w = await mountFill({ preview: true });

            await w.find("form").trigger("submit");
            await flushPromises();

            expect(formService.submitPublicForm).not.toHaveBeenCalled();
            expect(w.text()).not.toContain("Thank you");
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

    it("shows the form's cover at its position and its logo", async () => {
        formService.getPublicForm.mockResolvedValue({
            data: {
                ...form,
                workspace_id: "w1",
                cover_image: "c1",
                cover_position: 20,
                logo: "l1",
            },
        });
        const w = await mountFill();

        const cover = w.find("img[alt='Form cover image']");
        const logo = w.find("img[alt='Form logo']");

        expect(cover.attributes("src")).toBe("/api/workspaces/w1/file/c1");
        expect(cover.attributes("style")).toContain("50% 20%");
        expect(logo.attributes("src")).toBe("/api/workspaces/w1/file/l1");
        expect(w.text()).not.toContain("Reposition");
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
