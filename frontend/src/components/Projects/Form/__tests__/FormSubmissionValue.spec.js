// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormSubmissionValue from "@/components/Projects/Form/FormSubmissionValue.vue";

function show(field, value) {
    return mount(FormSubmissionValue, { props: { field, value } });
}

describe("FormSubmissionValue", () => {
    beforeEach(() => setActivePinia(createPinia()));

    it.each([
        ["an empty answer", { field_type: "text" }, ""],
        ["an unticked checkbox", { field_type: "bool" }, 0],
        ["no linked rows", { linked_id: "t2" }, []],
    ])("shows a dash for %s", (_, field, value) => {
        expect(show(field, value).text()).toBe("–");
    });

    it("shows a ticked checkbox as a tick", () => {
        expect(show({ field_type: "bool" }, 1).find("svg").exists()).toBe(true);
    });

    it("names linked rows and hides those the user may not see", () => {
        const w = show({ linked_id: "t2" }, [
            { id: "r1", name: "Acme" },
            { id: "r2", restricted: true },
        ]);

        expect(w.text()).toContain("Acme");
        expect(w.text()).toContain("Hidden");
    });

    it("counts files", () => {
        expect(show({ field_type: "file" }, [{ id: "a" }, { id: "b" }]).text()).toBe("2 files");
    });

    it("shows a day as the user writes dates", () => {
        expect(show({ field_type: "date" }, "2026-11-15").text()).toBe("15.11.2026");
    });
});
