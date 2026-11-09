// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, describe, expect, it } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import TableIconPicker from "@/components/Projects/TableIconPicker.vue";

describe("TableIconPicker", () => {
    let w;

    afterEach(() => w?.unmount());

    async function openPicker(icon = "") {
        w = mount(TableIconPicker, { props: { icon }, attachTo: document.body });
        await w.find("button").trigger("click");
        await flushPromises();
    }

    it("names every icon", async () => {
        await openPicker();

        const star = document.querySelector('[aria-label="Star"]');

        expect(star).not.toBeNull();
        expect(document.querySelectorAll("[aria-pressed]").length).toBe(25);
    });

    it("emits the chosen icon", async () => {
        await openPicker();
        document.querySelector('[aria-label="Star"]').click();
        await flushPromises();

        expect(w.emitted("select")).toEqual([["StarIcon"]]);
    });

    it("does not emit the icon already shown", async () => {
        await openPicker("StarIcon");
        document.querySelector('[aria-label="Star"]').click();
        await flushPromises();

        expect(w.emitted("select")).toBeUndefined();
    });

    it("marks the default table icon when none is set", async () => {
        await openPicker();

        expect(document.querySelector('[aria-label="Table"]').getAttribute("aria-pressed")).toBe(
            "true",
        );
    });
});
