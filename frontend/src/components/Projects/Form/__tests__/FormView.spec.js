// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { h } from "vue";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter, RouterView } from "vue-router";
// Registers the time zones the submissions table formats dates with.
import "moment-timezone";
import ConfirmDialog from "@/components/ConfirmDialog.vue";
import FormView from "@/components/Projects/Form/FormView.vue";
import FormViewNameDialog from "@/components/Projects/Form/FormViewNameDialog.vue";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { useAlertStore } from "@/store/alerts";
import { useSettingsStore } from "@/store/settings";
import { useWorkspaceRolesStore } from "@/store/workspaceRoles";

vi.mock("@/services/formService", () => ({
    default: {
        getForm: vi.fn(),
        createForm: vi.fn(),
        updateForm: vi.fn(),
        getPublicForm: vi.fn(),
    },
}));
vi.mock("@/services/workspaceService", () => ({
    default: { getWorkspaceNav: vi.fn() },
}));

const nav = {
    tables: [
        {
            id: "t1",
            headers: [
                { id: "h-name", name: "name" },
                { id: "h-budget", name: "budget", display_name: { String: "Budget" } },
            ],
        },
    ],
};

const viewPermissions = ["create_workspace_view", "update_workspace_view"];

async function mountAt(path, { licensed = true, user = { permissions: viewPermissions } } = {}) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setLicense({ forms: licensed });
    useWorkspaceRolesStore().setUser(user);

    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: "/p/:id/form/:tid", name: "form-view-create", component: FormView },
            { path: "/p/:id/form/:tid/view/:fid", name: "form-view", component: FormView },
            { path: "/forms/public/:token", name: "public-form", component: FormView },
        ],
    });

    await router.push(path);
    await router.isReady();

    const w = mount(FormView, { global: { plugins: [pinia, router] }, attachTo: document.body });

    await flushPromises();

    return { w, router };
}

async function mountRouted(path) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setLicense({ forms: true });
    useWorkspaceRolesStore().setUser({ permissions: viewPermissions });

    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: "/p/:id/form/:tid/view/:fid", name: "form-view", component: FormView },
            { path: "/elsewhere", name: "elsewhere", component: { render: () => null } },
        ],
    });

    await router.push(path);

    const w = mount(
        { render: () => h(RouterView) },
        { global: { plugins: [pinia, router] }, attachTo: document.body },
    );

    await flushPromises();

    return { w, router };
}

function button(w, label) {
    return w.findAll("button").find((b) => b.text() === label);
}

describe("FormView", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.stubGlobal(
            "ResizeObserver",
            class {
                observe() {}
                unobserve() {}
                disconnect() {}
            },
        );
        workspaceService.getWorkspaceNav.mockResolvedValue({ data: nav });
    });

    it("asks for a licence before loading anything", async () => {
        const { w } = await mountAt("/p/w1/form/t1/view/f1", { licensed: false });

        expect(w.text()).toContain("Build forms on your workspace tables");
        expect(formService.getForm).not.toHaveBeenCalled();
        w.unmount();
    });

    it("lists the table's fields for a new form", async () => {
        const { w } = await mountAt("/p/w1/form/t1");

        expect(formService.getForm).not.toHaveBeenCalled();
        expect(w.text()).toContain("New form");
        expect(w.text()).toContain("Budget");
        w.unmount();
    });

    it("loads a saved form", async () => {
        formService.getForm.mockResolvedValue({
            data: {
                id: "f1",
                name: "Intake",
                fields: [{ header_name: "budget", label: "Cost", visible: true, sort_order: 0 }],
            },
        });

        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        expect(formService.getForm).toHaveBeenCalledWith("w1", "t1", "f1");
        expect(w.text()).toContain("Intake");
        expect(w.text()).toContain("Cost");
        w.unmount();
    });

    it("explains a missing permission instead of showing an error", async () => {
        formService.getForm.mockRejectedValue({ response: { status: 403 } });

        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        expect(w.text()).toContain("You need permission to edit views");
        expect(useAlertStore().show).toBeFalsy();
        w.unmount();
    });

    it("says a deleted form no longer exists", async () => {
        formService.getForm.mockRejectedValue({ response: { status: 404 } });

        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        expect(w.text()).toContain("This form no longer exists");
        expect(w.text()).not.toContain("Something went wrong");
        w.unmount();
    });

    it("sends the version it opened and offers a reload when someone saved meanwhile", async () => {
        formService.getForm.mockResolvedValue({
            data: { id: "f1", name: "Intake", updated_at: 1700, fields: [] },
        });
        formService.updateForm.mockRejectedValue({ response: { status: 409 } });
        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        await w.find("input[type=text]").setValue("Signup");
        await button(w, "Update").trigger("click");
        await flushPromises();

        expect(formService.updateForm).toHaveBeenCalledWith(
            "w1",
            "t1",
            "f1",
            expect.objectContaining({ updated_at: 1700 }),
            expect.any(Array),
        );
        expect(w.text()).toContain("This form was changed by someone else");
        expect(useAlertStore().show).toBeFalsy();

        await button(w, "Reload").trigger("click");
        await flushPromises();

        expect(formService.getForm).toHaveBeenCalledTimes(2);
        expect(w.text()).not.toContain("This form was changed by someone else");
        w.unmount();
    });

    it("asks before leaving a form with unsaved changes", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Intake", fields: [] } });
        const { w, router } = await mountRouted("/p/w1/form/t1/view/f1");

        router.push("/elsewhere");
        await flushPromises();
        expect(router.currentRoute.value.name).toBe("elsewhere");

        await router.push("/p/w1/form/t1/view/f1");
        await flushPromises();
        await w.find("input[type=text]").setValue("Signup");

        router.push("/elsewhere");
        await flushPromises();

        const dialog = w.findComponent(ConfirmDialog);

        expect(dialog.props("open")).toBe(true);
        expect(router.currentRoute.value.name).toBe("form-view");

        dialog.vm.$emit("confirm");
        await flushPromises();
        expect(router.currentRoute.value.name).toBe("elsewhere");
        w.unmount();
    });

    it("warns the browser before a reload loses changes", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Intake", fields: [] } });
        const { w } = await mountAt("/p/w1/form/t1/view/f1");
        const unload = () => {
            const event = new Event("beforeunload", { cancelable: true });

            window.dispatchEvent(event);

            return event.defaultPrevented;
        };

        expect(unload()).toBe(false);

        await w.find("input[type=text]").setValue("Signup");

        expect(unload()).toBe(true);
        w.unmount();
        expect(unload()).toBe(false);
    });

    it("creates a form under the name given and opens it", async () => {
        formService.createForm.mockResolvedValue({ data: { id: "f9", name: "Intake" } });
        const { w, router } = await mountAt("/p/w1/form/t1");

        await button(w, "Save").trigger("click");
        w.findComponent(FormViewNameDialog).vm.$emit("confirm", "Intake");
        await flushPromises();

        expect(formService.createForm).toHaveBeenCalledWith(
            "w1",
            "t1",
            expect.objectContaining({ name: "Intake" }),
            expect.any(Array),
        );
        expect(router.currentRoute.value.params.fid).toBe("f9");
        expect(w.findComponent(FormViewNameDialog).props("open")).toBe(false);
        w.unmount();
    });

    it("updates a saved form with the edited title", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Intake", fields: [] } });
        formService.updateForm.mockResolvedValue({ data: { id: "f1", name: "Intake" } });
        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        await w.find("input[type=text]").setValue("Signup");
        await button(w, "Update").trigger("click");
        await flushPromises();

        expect(formService.updateForm).toHaveBeenCalledWith(
            "w1",
            "t1",
            "f1",
            expect.objectContaining({ title: "Signup" }),
            expect.any(Array),
        );
        w.unmount();
    });

    it("offers no save to a member who may not create or change views", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Intake", fields: [] } });

        const created = await mountAt("/p/w1/form/t1", { user: { permissions: ["create_task"] } });

        expect(button(created.w, "Save")).toBeUndefined();
        created.w.unmount();

        const saved = await mountAt("/p/w1/form/t1/view/f1", {
            user: { permissions: ["create_task"] },
        });

        expect(button(saved.w, "Update")).toBeUndefined();
        expect(button(saved.w, "Save as new")).toBeUndefined();
        saved.w.unmount();
    });

    it("lets a per-table role that manages the table's views save", async () => {
        const { w } = await mountAt("/p/w1/form/t1", {
            user: {
                permissions: ["create_task"],
                per_table_mode: true,
                table_permissions: [{ table_id: "t1", action: "manage_views" }],
            },
        });

        expect(button(w, "Save")).toBeDefined();
        w.unmount();
    });

    it("adds a per-table role that manages the table's views to a workspace role", async () => {
        const { w } = await mountAt("/p/w1/form/t1", {
            user: {
                permissions: ["create_task"],
                per_table_mode: false,
                table_permissions: [{ table_id: "t1", action: "manage_views" }],
            },
        });

        expect(button(w, "Save")).toBeDefined();
        w.unmount();
    });

    describe("a saved form", () => {
        const saved = {
            id: "f1",
            workspace_id: "w1",
            table_id: "t1",
            name: "Intake",
            public_token: "tok",
            fields: [{ header_name: "name", label: "Title", visible: true, sort_order: 0 }],
        };

        function tab(w, name) {
            return w.findAll("nav button").find((b) => b.text() === name);
        }

        beforeEach(() => {
            formService.getForm.mockResolvedValue({ data: saved });
        });

        it("previews the form as its link serves it", async () => {
            formService.getPublicForm.mockResolvedValue({
                data: {
                    ...saved,
                    title: "Tell us",
                    can_submit: true,
                    fields: [{ ...saved.fields[0], field_type: "text" }],
                },
            });
            const { w } = await mountAt("/p/w1/form/t1/view/f1");

            await tab(w, "Preview").trigger("click");
            await flushPromises();

            expect(formService.getPublicForm).toHaveBeenCalledWith("tok");
            expect(w.text()).toContain("Tell us");
            expect(w.find("button[type='submit']").exists()).toBe(true);
            w.unmount();
        });

        it("alerts when the link cannot be copied", async () => {
            vi.stubGlobal(
                "ResizeObserver",
                class {
                    observe() {}
                    disconnect() {}
                },
            );
            Object.assign(navigator, {
                clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
            });
            const { w } = await mountAt("/p/w1/form/t1/view/f1");
            const showError = vi.spyOn(useAlertStore(), "showError");

            await w
                .findAll("button")
                .find((b) => b.text() === "Form link")
                .trigger("click");
            await flushPromises();
            expect(document.body.querySelector("input[readonly]").value).toBe(
                `${window.location.origin}/forms/public/tok`,
            );

            [...document.body.querySelectorAll("button")]
                .find((b) => b.textContent.trim() === "Copy")
                .click();
            await flushPromises();

            expect(navigator.clipboard.writeText).toHaveBeenCalled();
            expect(showError).toHaveBeenCalledWith("Failed to copy link.");
            w.unmount();
        });
    });
});
