// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, nextTick } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { close, initialize } from "@/js/websocket";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { usePublicForm } from "../usePublicForm";

let deliver;

vi.mock("@/js/websocket", () => ({
    initialize: vi.fn(),
    close: vi.fn(),
    onProjectChange: (listener) => {
        deliver = listener;

        return () => {};
    },
}));

vi.mock("@/services/formService", () => ({
    default: { getPublicForm: vi.fn(), submitPublicForm: vi.fn() },
}));

const form = {
    id: "f1",
    title: "Intake",
    fields: [
        { header_name: "name", field_type: "text", visible: true, required: true },
        { header_name: "status", linked_id: "t-s", single_select: true, visible: true },
        { header_name: "secret", field_type: "text", visible: false, required: true },
    ],
};

function httpError(status, error = "Nope") {
    return Object.assign(new Error(error), { response: { status, data: { error } } });
}

function deferred() {
    let resolve;
    const promise = new Promise((r) => (resolve = r));

    return { promise, resolve };
}

async function setup(options) {
    setActivePinia(createPinia());

    let state;
    const Host = defineComponent({
        setup() {
            state = usePublicForm("tok", options);

            return {};
        },
        template: "<div/>",
    });
    const wrapper = mount(Host);

    await flushPromises();

    return { state, wrapper };
}

describe("usePublicForm", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        formService.getPublicForm.mockResolvedValue({ data: form });
    });

    it("loads the form and lists its visible fields", async () => {
        const { state } = await setup();

        expect(formService.getPublicForm).toHaveBeenCalledWith("tok");
        expect(state.status.value).toBe("ready");
        expect(state.fields.value.map((f) => f.header_name)).toEqual(["name", "status"]);
    });

    it.each([
        [401, "login"],
        [403, "forbidden"],
        [404, "not_found"],
        [500, "failed"],
    ])("shows a %i as %s", async (code, status) => {
        formService.getPublicForm.mockRejectedValue(httpError(code));

        const { state } = await setup();

        expect(state.status.value).toBe(status);
        expect(state.loadError.value).toBe("Nope");
    });

    it("does not send a form with a required field left empty", async () => {
        const { state } = await setup();

        await state.submit();

        expect(formService.submitPublicForm).not.toHaveBeenCalled();
        expect(Object.keys(state.fieldErrors.value)).toEqual(["name"]);
        expect(state.submitted.value).toBe(false);
    });

    it("does not send a decimal in a whole number field, and says what it takes", async () => {
        formService.getPublicForm.mockResolvedValue({
            data: {
                ...form,
                fields: [{ header_name: "count", field_type: "number", visible: true }],
            },
        });
        const { state } = await setup();

        state.values.value = { count: "1.5" };
        await state.submit();

        expect(formService.submitPublicForm).not.toHaveBeenCalled();
        expect(state.fieldErrors.value).toEqual({
            count: "Enter a whole number, such as 12, without a decimal part.",
        });

        state.values.value = { count: "2" };
        await nextTick();

        expect(state.fieldErrors.value).toEqual({});
    });

    it("says the form could not be loaded without a connection", async () => {
        formService.getPublicForm.mockRejectedValue(new Error("Network Error"));
        const { state } = await setup();

        expect(state.status.value).toBe("failed");
        expect(state.loadError.value).toBe(
            "Could not reach the server. Check your connection and try again.",
        );
    });

    it("clears a required field's error once it is filled in", async () => {
        const { state } = await setup();

        await state.submit();
        state.values.value = { ...state.values.value, name: "Order" };
        await nextTick();

        expect(state.fieldErrors.value).toEqual({});
    });

    it("sends the visible fields with the honeypot and thanks the user", async () => {
        formService.submitPublicForm.mockResolvedValue({ data: {} });
        const { state } = await setup();

        state.values.value = { name: "Task", status: { id: "s1", name: "Open" }, secret: "x" };
        await state.submit();

        expect(formService.submitPublicForm).toHaveBeenCalledWith(
            "tok",
            [
                { name: "name", value: "Task" },
                { name: "status", value: "s1" },
            ],
            "",
        );
        expect(state.submitted.value).toBe(true);

        state.startOver();
        expect(state.submitted.value).toBe(false);
        expect(state.values.value).toEqual({});
    });

    it("shows the server's reason a submit failed", async () => {
        formService.submitPublicForm.mockRejectedValue(httpError(400, "Dates reversed"));
        const { state } = await setup();

        state.values.value = { name: "Task" };
        await state.submit();

        expect(state.submitError.value).toBe("Dates reversed");
        expect(state.submitting.value).toBe(false);
        expect(state.submitted.value).toBe(false);
    });

    it("drops the server's reason once an answer changes", async () => {
        formService.submitPublicForm.mockRejectedValue(httpError(400, "Dates reversed"));
        const { state } = await setup();

        state.values.value = { name: "Task" };
        await state.submit();
        state.values.value.name = "Task, dates fixed";
        await nextTick();

        expect(state.submitError.value).toBe("");
    });

    describe("live", () => {
        it("connects while mounted", async () => {
            const { wrapper } = await setup({ live: true });

            expect(initialize).toHaveBeenCalled();
            wrapper.unmount();
            expect(close).toHaveBeenCalled();
        });

        it("reloads only when its own form changed", async () => {
            const { state } = await setup({ live: true });

            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "other" });
            deliver({ type: "task_created", form_id: "f1" });
            await flushPromises();
            expect(formService.getPublicForm).toHaveBeenCalledTimes(1);
            expect(state.configChanged.value).toBe(false);

            formService.getPublicForm.mockResolvedValue({ data: { ...form, title: "New" } });
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            expect(state.form.value.title).toBe("New");
            expect(state.configChanged.value).toBe(true);
        });

        it("keeps the latest of several quick changes", async () => {
            const { state } = await setup({ live: true });
            const older = deferred();

            formService.getPublicForm
                .mockReturnValueOnce(older.promise)
                .mockResolvedValueOnce({ data: { ...form, title: "Newest" } });
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            older.resolve({ data: { ...form, title: "Older" } });
            await flushPromises();

            expect(state.form.value.title).toBe("Newest");
        });

        it("drops the error of a field the change removed", async () => {
            const { state } = await setup({ live: true });

            await state.submit();
            expect(Object.keys(state.fieldErrors.value)).toEqual(["name"]);

            formService.getPublicForm.mockResolvedValue({
                data: { ...form, fields: form.fields.slice(1) },
            });
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            expect(state.fieldErrors.value).toEqual({});
        });

        it("keeps the answers of a form switched off meanwhile and sends none", async () => {
            const { state } = await setup({ live: true });

            state.values.value.name = "Ann";

            formService.getPublicForm.mockRejectedValue(httpError(404));
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            expect(state.status.value).toBe("ready");
            expect(state.closed.value).toBe(true);
            expect(state.values.value.name).toBe("Ann");

            await state.submit();
            expect(formService.submitPublicForm).not.toHaveBeenCalled();
        });

        it("opens again with the answers kept when the form is switched back on", async () => {
            const { state } = await setup({ live: true });

            state.values.value.name = "Ann";

            formService.getPublicForm.mockRejectedValueOnce(httpError(404));
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            expect(state.closed.value).toBe(false);
            expect(state.values.value.name).toBe("Ann");

            formService.submitPublicForm.mockResolvedValue({});
            await state.submit();
            expect(formService.submitPublicForm).toHaveBeenCalled();
        });

        it("alerts when the reload fails", async () => {
            const { state } = await setup({ live: true });
            const alerts = useAlertStore();
            const showError = vi.spyOn(alerts, "showError");

            formService.getPublicForm.mockRejectedValue(httpError(500, "Down"));
            deliver({ type: "FORM_CONFIG_CHANGED", form_id: "f1" });
            await flushPromises();

            expect(showError).toHaveBeenCalledWith("Down");
            expect(state.status.value).toBe("ready");
        });
    });
});
