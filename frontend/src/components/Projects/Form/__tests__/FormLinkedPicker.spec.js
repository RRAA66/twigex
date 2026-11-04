// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import formService from "@/services/formService";

vi.mock("@/services/formService", () => ({
    default: { getPublicLinkedItems: vi.fn() },
}));

function page(items, hasMore = false) {
    return { data: { items, has_more: hasMore } };
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
        formService.getPublicLinkedItems.mockResolvedValue(page([one]));
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

    it("shows the list when the field is clicked, not only from its button", async () => {
        formService.getPublicLinkedItems.mockResolvedValue(page([one, two]));
        w = mountPicker();
        await w.find("input").trigger("click");
        await flushPromises();

        expect(document.body.textContent).toContain("One");
        expect(document.body.textContent).toContain("Two");
    });

    it("adds the next page to the list", async () => {
        formService.getPublicLinkedItems
            .mockResolvedValueOnce(page([one], true))
            .mockResolvedValueOnce(page([two]));
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
            .mockResolvedValueOnce(page([one, two]))
            .mockReturnValueOnce(slow.promise)
            .mockResolvedValueOnce(page([two]));
        w = mountPicker();
        await open(w);

        const input = w.find("input");

        await input.setValue("o");
        vi.advanceTimersByTime(250);
        await input.setValue("tw");
        vi.advanceTimersByTime(250);
        await flushPromises();

        slow.resolve(page([one]));
        await flushPromises();

        expect(formService.getPublicLinkedItems).toHaveBeenLastCalledWith("tok", "t-x", {
            page: 1,
            limit: 50,
            search: "tw",
        });
        const options = [...document.body.querySelectorAll("[role='option']")];

        expect(options.map((o) => o.textContent.trim())).toEqual(["Two"]);
    });

    it("says when the options could not be loaded", async () => {
        formService.getPublicLinkedItems.mockRejectedValue(new Error("down"));
        w = mountPicker();
        await open(w);

        expect(document.body.textContent).toContain("Failed to load options.");
    });

    it("picks a row in a multiple pick that has no value yet", async () => {
        formService.getPublicLinkedItems.mockResolvedValue(page([one, two]));
        w = mountPicker({ multiple: true });
        await open(w);

        document
            .querySelector("[role=option]")
            .dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
        await flushPromises();

        expect(w.emitted("update:modelValue")[0]).toEqual([[one]]);
    });

    it("clears a single pick", async () => {
        w = mountPicker({ modelValue: one });

        await w.find("button:not([id^='headlessui'])").trigger("click");

        expect(w.emitted("update:modelValue")[0]).toEqual([null]);
    });

    it("offers no clearing before anything is picked", () => {
        w = mountPicker();

        expect(w.text()).not.toContain("Clear");
    });

    it("shows each option's colour and the picked one's", async () => {
        const high = { id: "o1", name: "High", color: "#dc2626" };

        formService.getPublicLinkedItems.mockResolvedValue(page([high, one]));
        w = mountPicker({ modelValue: high });
        await open(w);

        const dots = [...document.querySelectorAll("span.rounded-full")];

        expect(dots.map((d) => d.style.backgroundColor)).toEqual([
            "rgb(220, 38, 38)",
            "rgb(220, 38, 38)",
        ]);
    });

    it("removes a chosen row from a multiple pick", async () => {
        w = mountPicker({ multiple: true, modelValue: [one, two] });

        await w.find("button[type='button']").trigger("click");

        expect(w.emitted("update:modelValue")[0]).toEqual([[two]]);
    });
});
