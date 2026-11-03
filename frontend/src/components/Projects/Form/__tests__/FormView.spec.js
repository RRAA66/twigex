// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter } from "vue-router";
import FormView from "@/components/Projects/Form/FormView.vue";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { useAlertStore } from "@/store/alerts";
import { useSettingsStore } from "@/store/settings";

vi.mock("@/services/formService", () => ({
    default: { getForm: vi.fn(), createForm: vi.fn(), updateForm: vi.fn() },
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

async function mountAt(path, { licensed = true } = {}) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setLicense({ forms: licensed });

    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: "/p/:id/form/:tid", name: "form-view-create", component: FormView },
            { path: "/p/:id/form/:tid/view/:fid", name: "form-view", component: FormView },
        ],
    });

    await router.push(path);
    await router.isReady();

    const w = mount(FormView, { global: { plugins: [pinia, router] }, attachTo: document.body });

    await flushPromises();

    return w;
}

describe("FormView", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        workspaceService.getWorkspaceNav.mockResolvedValue({ data: nav });
    });

    it("asks for a licence before loading anything", async () => {
        const w = await mountAt("/p/w1/form/t1/view/f1", { licensed: false });

        expect(w.text()).toContain("Build forms on your workspace tables");
        expect(formService.getForm).not.toHaveBeenCalled();
        w.unmount();
    });

    it("lists the table's fields for a new form", async () => {
        const w = await mountAt("/p/w1/form/t1");

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

        const w = await mountAt("/p/w1/form/t1/view/f1");

        expect(formService.getForm).toHaveBeenCalledWith("w1", "t1", "f1");
        expect(w.text()).toContain("Intake");
        expect(w.text()).toContain("Cost");
        w.unmount();
    });

    it("explains a missing permission instead of showing an error", async () => {
        formService.getForm.mockRejectedValue({ response: { status: 403 } });

        const w = await mountAt("/p/w1/form/t1/view/f1");

        expect(w.text()).toContain("You need permission to edit views");
        expect(useAlertStore().show).toBeFalsy();
        w.unmount();
    });
});
