// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter } from "vue-router";
import ConfirmDialog from "@/components/ConfirmDialog.vue";
import ViewActions from "@/components/Projects/Menu/ViewActions.vue";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { useWorkspaceStore } from "@/store/workspaces";

vi.mock("@/services/formService", () => ({
    default: { deleteForm: vi.fn() },
}));
vi.mock("@/services/workspaceService", () => ({
    default: { deleteWorkspaceTableView: vi.fn() },
}));

async function mountAt(path) {
    const pinia = createPinia();

    setActivePinia(pinia);

    const Stub = { template: "<div />" };
    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: "/p/:id/grid/:tid", name: "grid-view", component: Stub },
            { path: "/p/:id/form/:tid/view/:fid", name: "form-view", component: Stub },
        ],
    });

    await router.push(path);
    await router.isReady();

    const w = mount(ViewActions, {
        global: { plugins: [pinia, router], stubs: { ConfirmDialog: true, RenameDialog: true } },
    });

    return { w, router };
}

async function confirmDelete(w, view) {
    w.vm.remove(view);
    await w.vm.$nextTick();
    w.findComponent(ConfirmDialog).vm.$emit("confirm");
    await flushPromises();
}

describe("ViewActions", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        formService.deleteForm.mockResolvedValue({});
        workspaceService.deleteWorkspaceTableView.mockResolvedValue({});
    });

    it("deletes a form's tab through the form and leaves the open form", async () => {
        const { w, router } = await mountAt("/p/w1/form/t1/view/f1");
        const workspaceStore = useWorkspaceStore();

        workspaceStore.setViewTypes([
            { id: "f1", view_type: "form" },
            { id: "g1", view_type: "grid" },
        ]);

        await confirmDelete(w, { id: "f1", view_type: "form" });

        expect(formService.deleteForm).toHaveBeenCalledWith("w1", "t1", "f1");
        expect(workspaceService.deleteWorkspaceTableView).not.toHaveBeenCalled();
        expect(workspaceStore.getViewTypes.map((v) => v.id)).toEqual(["g1"]);
        expect(router.currentRoute.value.name).toBe("grid-view");
    });

    it("says before deleting a form that the rows it added stay", async () => {
        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        w.vm.remove({ id: "f1", view_type: "form" });
        await w.vm.$nextTick();

        expect(w.findComponent(ConfirmDialog).props("message")).toContain("stay in the table");

        w.vm.remove({ id: "k1", view_type: "kanban" });
        await w.vm.$nextTick();

        expect(w.findComponent(ConfirmDialog).props("message")).not.toContain("stay in the table");
    });

    it("deletes any other tab as a view", async () => {
        const { w } = await mountAt("/p/w1/form/t1/view/f1");

        await confirmDelete(w, { id: "k1", view_type: "kanban" });

        expect(workspaceService.deleteWorkspaceTableView).toHaveBeenCalledWith({
            workspace_id: "w1",
            table_id: "t1",
            view_id: "k1",
        });
        expect(formService.deleteForm).not.toHaveBeenCalled();
    });
});
