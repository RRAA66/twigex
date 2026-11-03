// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormConditionValueInput from "@/components/Projects/Form/FormConditionValueInput.vue";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import userService from "@/services/userService";
import workspaceService from "@/services/workspaceService";

vi.mock("@/services/workspaceService", () => ({
    default: {
        getTableStatusTypes: vi.fn(),
        getLinkedRecordsLite: vi.fn(),
        searchWorkspaceMembers: vi.fn(),
    },
}));

vi.mock("@/services/userService", () => ({
    default: { byIds: vi.fn() },
}));

const client = { header_name: "client", field_type: "link", linked_id: "t-clients" };
const assignee = { header_name: "assignee", field_type: "default_assignee" };

function mountInput(props) {
    const pinia = createPinia();

    setActivePinia(pinia);

    return mount(FormConditionValueInput, {
        props: { workspaceId: "ws", tableId: "t1", label: "Value", ...props },
        global: { plugins: [pinia] },
        attachTo: document.body,
    });
}

function picker(w) {
    return w.findComponent(FormLinkedPicker);
}

describe("FormConditionValueInput", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("pages a linked table's rows through the workspace", async () => {
        const rows = Array.from({ length: 51 }, (_, i) => ({ id: i, name: `Row ${i}` }));

        workspaceService.getLinkedRecordsLite.mockResolvedValue({ data: rows });
        const w = mountInput({ field: client });

        const page = await picker(w).props("loader")({ page: 2, limit: 50, search: "ro" });

        expect(workspaceService.getLinkedRecordsLite).toHaveBeenCalledWith("ws", "t-clients", {
            q: "ro",
            limit: 51,
            offset: 50,
        });
        expect(page.items).toHaveLength(50);
        expect(page.items[0]).toEqual({ id: "0", name: "Row 0" });
        expect(page.has_more).toBe(true);
        w.unmount();
    });

    it("searches the workspace's members on the server", async () => {
        workspaceService.searchWorkspaceMembers.mockResolvedValue({
            data: [{ id: "u1", name: "Ann", lastname: "Lee" }],
        });
        const w = mountInput({ field: assignee });

        const page = await picker(w).props("loader")({ page: 3, limit: 50, search: "an" });

        expect(workspaceService.searchWorkspaceMembers).toHaveBeenCalledWith("ws", "an", 51, 100);
        expect(page).toEqual({ items: [{ id: "u1", name: "Ann Lee" }], has_more: false });

        await picker(w).vm.$emit("update:modelValue", page.items[0]);

        expect(w.emitted("update:modelValue")[0]).toEqual(["u1"]);
        w.unmount();
    });

    it("shows the name of a saved member that no page has listed", async () => {
        userService.byIds.mockResolvedValue({
            data: [{ id: "u9", name: "Zed", lastname: "Ray" }],
        });
        const w = mountInput({ field: assignee, modelValue: "u9" });

        await flushPromises();

        expect(picker(w).props("modelValue")).toEqual({ id: "u9", name: "Zed Ray" });
        w.unmount();
    });

    it("shows the name of a saved row", async () => {
        workspaceService.getLinkedRecordsLite.mockResolvedValue({
            data: [{ id: 7, name: "Acme" }],
        });
        const w = mountInput({ field: client, modelValue: "7" });

        await flushPromises();

        expect(workspaceService.getLinkedRecordsLite).toHaveBeenCalledWith("ws", "t-clients", {
            ids: "7",
        });
        expect(picker(w).props("modelValue")).toEqual({ id: "7", name: "Acme" });
        w.unmount();
    });

    it("drops the picked row once the value is cleared", async () => {
        const w = mountInput({ field: client, modelValue: "" });

        await picker(w).vm.$emit("update:modelValue", { id: "7", name: "Acme" });
        await w.setProps({ modelValue: "7" });

        expect(picker(w).props("modelValue")).toEqual({ id: "7", name: "Acme" });
        expect(workspaceService.getLinkedRecordsLite).not.toHaveBeenCalled();

        await w.setProps({ operator: "is_empty", modelValue: "" });
        await w.setProps({ operator: "equals" });

        expect(picker(w).props("modelValue")).toBeNull();
        w.unmount();
    });
});
