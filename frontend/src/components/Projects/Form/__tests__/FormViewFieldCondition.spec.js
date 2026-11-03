// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { reactive } from "vue";
import FormViewFieldCondition from "@/components/Projects/Form/FormViewFieldCondition.vue";
import workspaceService from "@/services/workspaceService";

vi.mock("@/services/workspaceService", () => ({
    default: {
        getTableStatusTypes: vi.fn(),
        getLinkedRecordsLite: vi.fn(),
        searchWorkspaceMembers: vi.fn(),
    },
}));

function mountCondition(field, fields) {
    const pinia = createPinia();

    setActivePinia(pinia);

    return mount(FormViewFieldCondition, {
        props: { field, fields, workspaceId: "ws", tableId: "t1" },
        global: { plugins: [pinia] },
        attachTo: document.body,
    });
}

describe("FormViewFieldCondition", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("adds and removes the field's condition", async () => {
        const field = reactive({ header_name: "reason", visible: true, visibility_rule: null });
        const w = mountCondition(field, [field]);

        await w.find("input[type='checkbox']").setValue(true);

        expect(field.visibility_rule).toEqual({ trigger_field: "", operator: "equals", value: "" });

        await w.find("input[type='checkbox']").setValue(false);

        expect(field.visibility_rule).toBeNull();
        w.unmount();
    });

    it("picks a status value from the table's status options", async () => {
        workspaceService.getTableStatusTypes.mockResolvedValue({
            data: { options: [{ id: "s1", name: "Open" }] },
        });
        const status = {
            header_name: "status",
            label: "Status",
            visible: true,
            field_type: "status",
        };
        const field = reactive({
            header_name: "reason",
            visible: true,
            visibility_rule: { trigger_field: "status", operator: "equals", value: "s1" },
        });
        const w = mountCondition(field, [status, field]);

        await flushPromises();

        expect(workspaceService.getTableStatusTypes).toHaveBeenCalledWith("ws", "t1");
        expect(w.text()).toContain("Status");
        expect(w.text()).toContain("Open");
        w.unmount();
    });
});
