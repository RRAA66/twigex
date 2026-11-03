// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import formService from "@/services/formService";
import workspaceService from "@/services/workspaceService";

vi.mock("@/services/formService", () => ({
    default: { getPublicLinkedItems: vi.fn() },
}));

vi.mock("@/services/workspaceService", () => ({
    default: { getLinkedRecordsLite: vi.fn() },
}));

function page(items, total) {
    return { data: { items, total } };
}

function deferred() {
    let resolve;
    const promise = new Promise((r) => (resolve = r));

    return { promise, resolve };
}

const one = { id: "r1", name: "One" };
const two = { id: "r2", name: "Two" };

function mountPicker(props = {}) {
    return mount(FormLinkedPicker, {
        props: { token: "tok", tableId: "t-x", ...props },
        attachTo: document.body,
    });
}

async function open(w) {
    await w.find("input").trigger("focus");
    await w.find("button[id^='headlessui-combobox-button']").trigger("click");
    await flushPromises();
}

describe("FormLinkedPicker", () => {
    let w;

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
    });

    afterEach(() => {
        w?.unmount();
        vi.useRealTimers();
    });

    it("loads the first page only when opened", async () => {
        formService.getPublicLinkedItems.mockResolvedValue(page([one], 1));
        w = mountPicker();
        await flushPromises();

        expect(formService.getPublicLinkedItems).not.toHaveBeenCalled();

        await open(w);

        expect(formService.getPublicLinkedItems).toHaveBeenCalledTimes(1);
        expect(formService.getPublicLinkedItems).toHaveBeenCalledWith("tok", "t-x", {
            page: 1,
            limit: 50,
            search: undefined,
        });
        expect(document.body.textContent).toContain("One");
    });

    it("adds the next page to the list", async () => {
        formService.getPublicLinkedItems
            .mockResolvedValueOnce(page([one], 2))
            .mockResolvedValueOnce(page([two], 2));
        w = mountPicker();
        await open(w);

        const more = [...document.body.querySelectorAll("button")].find((b) =>
            b.textContent.includes("Load more"),
        );

        more.click();
        await flushPromises();

        expect(formService.getPublicLinkedItems).toHaveBeenLastCalledWith("tok", "t-x", {
            page: 2,
            limit: 50,
            search: undefined,
        });
        expect(document.body.textContent).toContain("One");
        expect(document.body.textContent).toContain("Two");
        expect(document.body.textContent).not.toContain("Load more");
    });

    it("searches after typing stops and ignores an older answer", async () => {
        const slow = deferred();

        formService.getPublicLinkedItems
            .mockResolvedValueOnce(page([one, two], 2))
            .mockReturnValueOnce(slow.promise)
            .mockResolvedValueOnce(page([two], 1));
        w = mountPicker();
        await open(w);

        const input = w.find("input");

        await input.setValue("o");
        vi.advanceTimersByTime(250);
        await input.setValue("tw");
        vi.advanceTimersByTime(250);
        await flushPromises();

        slow.resolve(page([one], 1));
        await flushPromises();

        expect(formService.getPublicLinkedItems).toHaveBeenLastCalledWith("tok", "t-x", {
            page: 1,
            limit: 50,
            search: "tw",
        });
        const options = [...document.body.querySelectorAll("[role='option']")];

        expect(options.map((o) => o.textContent.trim())).toEqual(["Two"]);
    });

    it("lists a workspace's rows when it has no form link", async () => {
        const rows = Array.from({ length: 50 }, (_, i) => ({ id: i, name: `Row ${i}` }));

        workspaceService.getLinkedRecordsLite.mockResolvedValue({ data: rows });
        w = mountPicker({ token: "", workspaceId: "ws" });
        await open(w);

        expect(formService.getPublicLinkedItems).not.toHaveBeenCalled();
        expect(workspaceService.getLinkedRecordsLite).toHaveBeenCalledWith("ws", "t-x", {
            q: undefined,
            limit: 50,
            offset: 0,
        });
        expect(document.body.textContent).toContain("Row 49");
        expect(document.body.textContent).toContain("Load more");
    });

    it("says when the options could not be loaded", async () => {
        formService.getPublicLinkedItems.mockRejectedValue(new Error("down"));
        w = mountPicker();
        await open(w);

        expect(document.body.textContent).toContain("Failed to load options.");
    });

    it("removes a chosen row from a multiple pick", async () => {
        w = mountPicker({ multiple: true, modelValue: [one, two] });

        await w.find("button[type='button']").trigger("click");

        expect(w.emitted("update:modelValue")[0]).toEqual([[two]]);
    });
});
