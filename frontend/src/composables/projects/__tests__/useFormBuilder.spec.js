// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createMemoryHistory, createRouter } from "vue-router";
import { useFormBuilder } from "@/composables/projects/useFormBuilder";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { useSettingsStore } from "@/store/settings";
import { useWorkspaceStore } from "@/store/workspaces";

vi.mock("@/services/formService", () => ({
    default: { getForm: vi.fn(), createForm: vi.fn(), updateForm: vi.fn() },
}));
vi.mock("@/services/workspaceService", () => ({
    default: { getWorkspaceNav: vi.fn() },
}));

const nav = {
    tables: [{ id: "t1", headers: [{ id: "h-name", name: "name" }] }],
};

function deferred() {
    let resolve;
    const promise = new Promise((r) => (resolve = r));

    return { promise, resolve };
}

async function mountBuilder(path) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setLicense({ forms: true });

    const Stub = { template: "<div />" };
    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: "/p/:id/form/:tid", name: "form-view-create", component: Stub },
            { path: "/p/:id/form/:tid/view/:fid", name: "form-view", component: Stub },
        ],
    });

    await router.push(path);
    await router.isReady();

    let builder;
    const Harness = defineComponent({
        setup() {
            builder = useFormBuilder();

            return () => null;
        },
    });

    mount(Harness, { global: { plugins: [pinia, router] } });
    await flushPromises();

    return { builder, router };
}

describe("useFormBuilder", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        workspaceService.getWorkspaceNav.mockResolvedValue({ data: nav });
    });

    it("opens the created form's own route", async () => {
        formService.createForm.mockResolvedValue({ data: { id: "f9", name: "Intake" } });
        const { builder, router } = await mountBuilder("/p/w1/form/t1");

        expect(await builder.createForm("Intake")).toBe(true);
        await flushPromises();

        expect(formService.createForm).toHaveBeenCalledWith(
            "w1",
            "t1",
            expect.objectContaining({ id: "", name: "Intake" }),
            [expect.objectContaining({ header_name: "name", sort_order: 0 })],
        );
        expect(router.currentRoute.value.name).toBe("form-view");
        expect(router.currentRoute.value.params.fid).toBe("f9");
        expect(formService.getForm).not.toHaveBeenCalled();
    });

    it("renames the form's tab after an update", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Old", fields: [] } });
        formService.updateForm.mockResolvedValue({ data: { id: "f1", name: "New" } });
        const { builder } = await mountBuilder("/p/w1/form/t1/view/f1");
        const workspaceStore = useWorkspaceStore();

        workspaceStore.setViewTypes([
            { id: "f1", name: "Old" },
            { id: "g1", name: "Grid" },
        ]);

        await builder.updateForm();

        expect(formService.updateForm).toHaveBeenCalledWith(
            "w1",
            "t1",
            "f1",
            expect.objectContaining({ id: "f1" }),
            expect.any(Array),
        );
        expect(workspaceStore.getViewTypes.map((v) => v.name)).toEqual(["New", "Grid"]);
        expect(builder.formData.value.name).toBe("New");
    });

    it("keeps the form of the latest route when an earlier load answers last", async () => {
        const first = deferred();
        const second = deferred();

        formService.getForm.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
        const { builder, router } = await mountBuilder("/p/w1/form/t1/view/f1");

        await router.push("/p/w1/form/t1/view/f2");
        await flushPromises();

        second.resolve({ data: { id: "f2", name: "Second", fields: [] } });
        await flushPromises();
        first.resolve({ data: { id: "f1", name: "First", fields: [] } });
        await flushPromises();

        expect(builder.formData.value).toMatchObject({ id: "f2", name: "Second" });
        expect(builder.status.value).toBe("ready");
    });

    it("reloads when the table changes under the same form route", async () => {
        formService.getForm.mockResolvedValue({ data: { id: "f1", name: "Intake", fields: [] } });
        const { router } = await mountBuilder("/p/w1/form/t1/view/f1");

        await router.push("/p/w1/form/t2/view/f1");
        await flushPromises();

        expect(formService.getForm).toHaveBeenLastCalledWith("w1", "t2", "f1");
    });

    it("changes a field and the order without touching the old objects", async () => {
        const { builder } = await mountBuilder("/p/w1/form/t1");
        const before = builder.formFields.value;

        builder.updateField("name", { label: "Title" });
        builder.updateSettings({ title: "Signup" });

        expect(builder.formFields.value[0].label).toBe("Title");
        expect(before[0].label).not.toBe("Title");
        expect(builder.formData.value.title).toBe("Signup");

        builder.reorderFields([]);

        expect(builder.formFields.value).toEqual([]);
    });
});
