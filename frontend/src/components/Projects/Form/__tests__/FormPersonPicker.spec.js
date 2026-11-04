// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormPersonPicker from "@/components/Projects/Form/FormPersonPicker.vue";
import workspaceService from "@/services/workspaceService";
import { useUserStore } from "@/store/user";

vi.mock("@/services/workspaceService", () => ({
    default: { searchWorkspaceMembers: vi.fn() },
}));

const ann = { id: "u1", name: "Ann", lastname: "Lee" };
const ben = { id: "u2", name: "Ben", lastname: "" };

function mountPicker(modelValue = "") {
    return mount(FormPersonPicker, {
        props: { workspaceId: "w1", modelValue },
        attachTo: document.body,
    });
}

async function open(w) {
    await w.find("input").trigger("click");
    await flushPromises();
}

describe("FormPersonPicker", () => {
    let w;

    beforeEach(() => {
        setActivePinia(createPinia());
        vi.clearAllMocks();
    });

    afterEach(() => w?.unmount());

    it("searches the workspace's members a page at a time", async () => {
        workspaceService.searchWorkspaceMembers.mockResolvedValue({ data: [ann, ben] });
        w = mountPicker();

        await open(w);

        expect(workspaceService.searchWorkspaceMembers).toHaveBeenCalledWith("w1", "", 51, 0);
        expect(document.body.textContent).toContain("Ann Lee");
        expect(document.body.textContent).toContain("Ben");
    });

    it("answers the picked member's id", async () => {
        workspaceService.searchWorkspaceMembers.mockResolvedValue({ data: [ann] });
        w = mountPicker();

        await open(w);
        document
            .querySelector("[role=option]")
            .dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
        await flushPromises();

        expect(w.emitted("update:modelValue")[0]).toEqual(["u1"]);
    });

    it("names a member picked before by their id", async () => {
        useUserStore().addUsers([ann]);
        w = mountPicker("u1");
        await flushPromises();

        expect(w.find("input").element.value).toBe("Ann Lee");
    });

    it("answers no one when cleared", async () => {
        useUserStore().addUsers([ann]);
        w = mountPicker("u1");
        await flushPromises();

        await w
            .findAll("button")
            .find((b) => b.text() === "Clear")
            .trigger("click");

        expect(w.emitted("update:modelValue")[0]).toEqual([""]);
    });
});
