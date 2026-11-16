// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { createRouter, createMemoryHistory } from "vue-router";
import { useSettingsStore } from "@/store/settings";
import { useWorkspaceStore } from "@/store/workspaces";
import { useWorkspaceRolesStore } from "@/store/workspaceRoles";
import TopNavigation from "@/components/Projects/TopNavigation.vue";

vi.mock("@/services/workspaceService", () => ({
    default: new Proxy({}, { get: () => () => new Promise(() => {}) }),
}));
globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
};

async function mountNavigation(formsOn) {
    const pinia = createPinia();

    setActivePinia(pinia);
    useSettingsStore().setConfig({ FeatureFlags: { Forms: formsOn } });
    useWorkspaceRolesStore().user = { permissions: ["create_workspace_view"] };
    useWorkspaceStore().setViewTypes([
        { id: "v1", name: "Grid View", view_type: "grid", is_default: true },
        { id: "f1", name: "Sign-up form", view_type: "form" },
    ]);

    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            {
                path: "/p/:id/grid/:tid/view/:fid?",
                name: "grid-view",
                component: { template: "<div/>" },
            },
            { path: "/:pathMatch(.*)*", component: { template: "<div/>" } },
        ],
    });

    await router.push("/p/w1/grid/t1/view/v1");
    await router.isReady();

    const wrapper = mount(TopNavigation, {
        global: { plugins: [pinia, router] },
        attachTo: document.body,
    });

    await flushPromises();
    const newView = wrapper.findAll("button").find((b) => b.text() === "New view");

    await newView.trigger("click");
    await flushPromises();

    return wrapper;
}

function newViewOptions(wrapper) {
    return wrapper.findAll("[role='menuitem']").map((item) => item.text());
}

describe("TopNavigation with the Forms flag", () => {
    it("hides form tabs and the Form view option when Forms is off", async () => {
        const wrapper = await mountNavigation(false);

        expect(wrapper.text()).not.toContain("Sign-up form");
        expect(newViewOptions(wrapper)).toHaveLength(2);

        wrapper.unmount();
    });

    it("shows form tabs and the Form view option when Forms is on", async () => {
        const wrapper = await mountNavigation(true);

        expect(wrapper.text()).toContain("Sign-up form");
        expect(newViewOptions(wrapper)).toHaveLength(3);

        wrapper.unmount();
    });
});
