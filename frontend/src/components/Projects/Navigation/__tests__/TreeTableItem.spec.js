// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { StarIcon, TableCellsIcon } from "@heroicons/vue/24/outline";
import TreeTableItem from "@/components/Projects/Navigation/TreeTableItem.vue";

vi.mock("vue-router", () => ({
    useRoute: () => ({ params: { id: "w1" } }),
    useRouter: () => ({ push: vi.fn() }),
}));

function iconPath(component) {
    return mount(component).find("path").attributes("d");
}

function mountItem(table) {
    setActivePinia(createPinia());

    return mount(TreeTableItem, { props: { table: { id: "t1", name: "Tasks", ...table } } });
}

describe("TreeTableItem", () => {
    it("shows the icon chosen for the table", () => {
        const w = mountItem({ icon: "StarIcon" });

        expect(w.find("path").attributes("d")).toBe(iconPath(StarIcon));
    });

    it("shows the table icon when none is chosen", () => {
        const w = mountItem({});

        expect(w.find("path").attributes("d")).toBe(iconPath(TableCellsIcon));
    });
});
