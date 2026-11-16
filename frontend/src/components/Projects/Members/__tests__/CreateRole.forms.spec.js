// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createRouter, createMemoryHistory } from "vue-router";
import { useSettingsStore } from "@/store/settings";
import CreateRole from "@/components/Projects/Members/CreateRole.vue";

vi.mock("@/services/workspaceService", () => ({
    default: { getTablesForRoleEditor: () => Promise.resolve({ data: { tables: [] } }) },
}));

async function viewTypeLabels(formsOn) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setConfig({ FeatureFlags: { Forms: formsOn } });

    const router = createRouter({
        history: createMemoryHistory(),
        routes: [{ path: "/:pathMatch(.*)*", component: { template: "<div/>" } }],
    });

    await router.push("/p/w1/roles/new");
    await router.isReady();

    const wrapper = mount(CreateRole, { global: { plugins: [pinia, router] } });

    await flushPromises();

    const labels = wrapper.findAll("label").map((label) => label.text());

    wrapper.unmount();

    return labels;
}

describe("CreateRole with the Forms flag", () => {
    it("hides the Form view type when Forms is off", async () => {
        const labels = await viewTypeLabels(false);

        expect(labels).toContain("Gantt");
        expect(labels).not.toContain("Form");
    });

    it("shows the Form view type when Forms is on", async () => {
        expect(await viewTypeLabels(true)).toContain("Form");
    });
});
