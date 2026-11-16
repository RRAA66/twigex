// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import DatePickerField from "@/components/DatePicker/DatePickerField.vue";

describe("DatePickerField", () => {
    it("moves the keyboard focus into the calendar it opens", async () => {
        const pinia = createPinia();

        setActivePinia(pinia);

        const w = mount(DatePickerField, {
            props: { id: "due" },
            global: { plugins: [pinia] },
            attachTo: document.body,
        });

        await w.find("#due").trigger("click");
        await flushPromises();
        await new Promise((resolve) => requestAnimationFrame(resolve));
        await flushPromises();

        const panel = document.body.querySelector("[id^='headlessui-popover-panel']");

        expect(panel).not.toBeNull();
        expect(panel.contains(document.activeElement)).toBe(true);
        w.unmount();
    });
});
