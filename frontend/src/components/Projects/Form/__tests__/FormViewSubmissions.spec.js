// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import "moment-timezone";
import FormViewSubmissions from "@/components/Projects/Form/FormViewSubmissions.vue";
import formService from "@/services/formService";
import { onProjectChange } from "@/js/websocket";

const openTask = vi.fn();

vi.mock("@/services/formService", () => ({
    default: { getSubmissions: vi.fn() },
}));
vi.mock("@/composables/projects/useOpenTask", () => ({
    useOpenTask: () => ({ openTask }),
}));
vi.mock("@/js/websocket", () => ({
    onProjectChange: vi.fn(() => () => {}),
}));

const fields = [
    { header_name: "name", label: "Title", field_type: "text", visible: true },
    {
        header_name: "status",
        label: "Status",
        field_type: "status",
        linked_id: "opts",
        single_select: true,
        visible: true,
    },
    { header_name: "notes", label: "Notes", field_type: "text", visible: false },
];

function submission(id, row) {
    return { id, task_id: `task-${id}`, submitted_by: "u1", submitted_at: 1700000000, row };
}

function respond(items, total = items.length) {
    formService.getSubmissions.mockResolvedValue({ data: { items, total } });
}

async function mountList() {
    const w = mount(FormViewSubmissions, {
        props: { workspaceId: "w1", tableId: "t1", formId: "f1", fields },
        attachTo: document.body,
    });

    await flushPromises();

    return w;
}

describe("FormViewSubmissions", () => {
    let w;

    beforeEach(() => {
        setActivePinia(createPinia());
        vi.clearAllMocks();
    });

    afterEach(() => {
        w?.unmount();
        vi.useRealTimers();
    });

    it("shows a column for each field on the form", async () => {
        respond([
            submission("1", {
                name: "Repair the roof",
                status: { id: "o1", name: "Open", color: "#16a34a" },
                notes: "secret",
            }),
        ]);
        w = await mountList();

        const headers = w.findAll("th").map((th) => th.text());

        expect(headers).toEqual(expect.arrayContaining(["Title", "Status"]));
        expect(headers).not.toContain("Notes");
        expect(w.text()).toContain("Repair the roof");
        expect(w.text()).toContain("Open");
        expect(w.text()).not.toContain("secret");
    });

    it("opens the row a submission made", async () => {
        respond([submission("1", { name: "Repair the roof" })]);
        w = await mountList();

        await w.find("tbody tr").trigger("click");

        expect(openTask).toHaveBeenCalledWith({
            workspaceId: "w1",
            tableId: "t1",
            taskId: "task-1",
        });
    });

    it("says when a row is gone and does not open it", async () => {
        respond([submission("1", undefined)]);
        w = await mountList();

        expect(w.text()).toContain("Row deleted or not visible to you");

        await w.find("tbody tr").trigger("click");

        expect(openTask).not.toHaveBeenCalled();
    });

    it("loads another page when one is picked", async () => {
        respond(
            Array.from({ length: 50 }, (_, i) => submission(String(i), { name: `Row ${i}` })),
            60,
        );
        w = await mountList();

        w.findComponent({ name: "DataTable" }).vm.$emit("update:currentPage", 2);
        await flushPromises();

        expect(formService.getSubmissions).toHaveBeenLastCalledWith("w1", "t1", "f1", 2);
    });

    it("reloads when a row of its table changes", async () => {
        vi.useFakeTimers();
        respond([]);
        w = await mountList();

        const listener = onProjectChange.mock.calls[0][0];

        listener({ type: "task_created", table_id: "other" });
        listener({ type: "task_created", table_id: "t1" });
        await vi.runAllTimersAsync();

        expect(formService.getSubmissions).toHaveBeenCalledTimes(2);
    });
});
