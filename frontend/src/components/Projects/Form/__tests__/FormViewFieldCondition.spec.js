// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import BaseSelect from "@/components/BaseSelect.vue";
import FormViewFieldCondition from "@/components/Projects/Form/FormViewFieldCondition.vue";
import workspaceService from "@/services/workspaceService";

vi.mock("@/services/workspaceService", () => ({
    default: {
        getTableStatusTypes: vi.fn(),
        getLinkedRecordsLite: vi.fn(),
        searchWorkspaceMembers: vi.fn(),
    },
}));

const status = { header_name: "status", label: "Status", visible: true, field_type: "status" };

function mountWith(component, props) {
    const pinia = createPinia();

    setActivePinia(pinia);

    return mount(component, {
        props: { workspaceId: "ws", tableId: "t1", ...props },
        global: { plugins: [pinia] },
        attachTo: document.body,
    });
}

describe("FormViewFieldCondition", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        workspaceService.getTableStatusTypes.mockResolvedValue({
            data: { options: [{ id: "s1", name: "Open" }] },
        });
    });

    it("emits the condition added and removed without changing the field", async () => {
        const field = Object.freeze({
            header_name: "reason",
            visible: true,
            visibility_rule: null,
        });
        const w = mountWith(FormViewFieldCondition, { field, fields: [field] });

        await w.find("input[type='checkbox']").setValue(true);

        expect(w.emitted("update")[0]).toEqual([
            { trigger_field: "", operator: "equals", value: "" },
        ]);

        await w.setProps({
            field: { ...field, visibility_rule: w.emitted("update")[0][0] },
        });
        await w.find("input[type='checkbox']").setValue(false);

        expect(w.emitted("update")[1]).toEqual([null]);
        w.unmount();
    });

    it("picks a status value from the table's status options", async () => {
        const field = {
            header_name: "reason",
            visible: true,
            visibility_rule: { trigger_field: "status", operator: "equals", value: "s1" },
        };
        const w = mountWith(FormViewFieldCondition, { field, fields: [status, field] });

        await flushPromises();

        expect(workspaceService.getTableStatusTypes).toHaveBeenCalledWith("ws", "t1");
        expect(w.text()).toContain("Status");
        expect(w.text()).toContain("Open");
        w.unmount();
    });

    it("labels the field, operator and value controls", async () => {
        const field = {
            header_name: "reason",
            visible: true,
            visibility_rule: { trigger_field: "status", operator: "equals", value: "" },
        };
        const w = mountWith(FormViewFieldCondition, { field, fields: [status, field] });

        await flushPromises();

        expect(w.findAll("label").map((l) => l.text())).toEqual(
            expect.arrayContaining(["When field", "Condition", "Value"]),
        );
        w.unmount();
    });

    it("clears the value when the operator takes another kind of input", async () => {
        const field = {
            header_name: "reason",
            visible: true,
            visibility_rule: { trigger_field: "status", operator: "equals", value: "s1" },
        };
        const w = mountWith(FormViewFieldCondition, { field, fields: [status, field] });

        await flushPromises();
        w.findAllComponents(BaseSelect)[1].vm.$emit("update:modelValue", "is_empty");

        expect(w.emitted("update")[0]).toEqual([
            { trigger_field: "status", operator: "is_empty", value: "" },
        ]);
        w.unmount();
    });
});
