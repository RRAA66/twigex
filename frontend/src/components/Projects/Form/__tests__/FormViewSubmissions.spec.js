// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import "moment-timezone";
import FormViewSubmissions from "@/components/Projects/Form/FormViewSubmissions.vue";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";
import { onProjectChange } from "@/js/websocket";
import { useUserStore } from "@/store/user";

const openTask = vi.fn();

vi.mock("@/services/formService", () => ({
    default: { getSubmissions: vi.fn() },
}));
vi.mock("@/services/workspaceService", () => ({
    default: { getWorkspaceTables: vi.fn() },
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

async function mountList(shown = fields) {
    const w = mount(FormViewSubmissions, {
        props: { workspaceId: "w1", tableId: "t1", formId: "f1", fields: shown },
        attachTo: document.body,
    });

    await flushPromises();

    return w;
}

describe("FormViewSubmissions", () => {
    let w;

    beforeEach(() => {
        setActivePinia(createPinia());
        useUserStore().setUser({ id: "me", timezone: { manualTimezone: "UTC" } });
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

    it("works a calculation out from the row, as it is never stored", async () => {
        respond([submission("1", { qty: "4", price: "12.5", total: null })]);
        w = await mountList([
            {
                header_name: "total",
                label: "Total",
                field_type: "calculations",
                visible: true,
                formula: {
                    name: "PRODUCT",
                    args: [
                        { type: "field", value: "qty" },
                        { type: "field", value: "price" },
                    ],
                },
            },
        ]);

        expect(w.find("tbody").text()).toContain("50");
    });

    it("names the table a master link points to", async () => {
        workspaceService.getWorkspaceTables.mockResolvedValue({
            data: [{ id: "t9", name: "clients", display_name: { Valid: true, String: "Clients" } }],
        });
        respond([submission("1", { which: "t9" })]);
        w = await mountList([
            { header_name: "which", label: "Which", field_type: "master link", visible: true },
        ]);

        expect(workspaceService.getWorkspaceTables).toHaveBeenCalledWith("w1", "nav");
        expect(w.text()).toContain("Clients");
    });

    it("loads no table names without a master link", async () => {
        respond([]);
        w = await mountList();

        expect(workspaceService.getWorkspaceTables).not.toHaveBeenCalled();
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

        expect(formService.getSubmissions).toHaveBeenLastCalledWith("w1", "t1", "f1", { page: 2 });
    });

    it("lists only the submissions sent on the days picked", async () => {
        respond([]);
        w = await mountList();

        const [from, to] = w.findAllComponents({ name: "DatePickerField" });

        from.vm.$emit("update:modelValue", "2026-11-03");
        to.vm.$emit("update:modelValue", "2026-11-04");
        await flushPromises();

        expect(formService.getSubmissions).toHaveBeenLastCalledWith("w1", "t1", "f1", {
            page: 1,
            from: Date.UTC(2026, 10, 3) / 1000,
            to: Date.UTC(2026, 10, 5) / 1000,
        });
    });

    it("warns about a From day after the To day and lists nothing for it", async () => {
        respond([submission("s1", { name: "A" })]);
        w = await mountList();
        formService.getSubmissions.mockClear();

        const [from, to] = w.findAllComponents({ name: "DatePickerField" });

        from.vm.$emit("update:modelValue", "2026-11-05");
        to.vm.$emit("update:modelValue", "2026-11-04");
        await flushPromises();

        expect(formService.getSubmissions).not.toHaveBeenCalled();
        expect(w.find("[role='alert']").text()).toBe(
            "The From day is after the To day. Pick a From day on or before it.",
        );
        expect(w.text()).toContain("No submissions in this date range");
        expect(w.findComponent({ name: "DataTable" }).props("items")).toEqual([]);

        const exportButton = w.findAll("button").find((b) => b.text() === "Export CSV");

        expect(exportButton.attributes("disabled")).toBeDefined();
    });

    it("says no submission was sent in the days picked", async () => {
        respond([]);
        w = await mountList();

        w.findAllComponents({ name: "DatePickerField" })[0].vm.$emit(
            "update:modelValue",
            "2026-11-03",
        );
        await flushPromises();

        expect(w.text()).toContain("No submissions in this date range");
    });

    it("downloads the submissions of the days picked as a CSV", async () => {
        respond([]);
        w = await mountList();

        w.findComponent({ name: "DatePickerField" }).vm.$emit("update:modelValue", "2026-11-03");
        await flushPromises();

        const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

        await w
            .findAll("button")
            .find((b) => b.text() === "Export CSV")
            .trigger("click");

        const href = new URL(click.mock.contexts[0].href);

        expect(href.pathname).toBe("/api/workspaces/w1/table/t1/forms/f1/submissions/export");
        expect(href.searchParams.get("from")).toBe(String(Date.UTC(2026, 10, 3) / 1000));
        expect(href.searchParams.get("tz")).toBe("UTC");
        expect(JSON.parse(href.searchParams.get("labels"))).toEqual({
            submitted_by: "Submitted by",
            submitted_at: "Date",
            "field:name": "Title",
            "field:status": "Status",
        });
        click.mockRestore();
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
