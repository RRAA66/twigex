// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, describe, expect, it } from "vitest";
import { nextTick } from "vue";
import { mount } from "@vue/test-utils";
import FormCoverImage from "@/components/Projects/Form/FormCoverImage.vue";
import FormViewSettings from "@/components/Projects/Form/FormViewSettings.vue";

function pickFile(input, file) {
    Object.defineProperty(input.element, "files", { value: [file], configurable: true });

    return input.trigger("change");
}

// jsdom has no PointerEvent; the handlers only read clientY.
function pointer(element, type, clientY) {
    element.dispatchEvent(new MouseEvent(type, { clientY, bubbles: true }));

    return nextTick();
}

describe("FormCoverImage", () => {
    it("moves the cover while repositioning and not before", async () => {
        const w = mount(FormCoverImage, {
            props: { src: "/c.png", position: 50, repositionable: true },
        });
        const frame = w.element;

        Object.defineProperty(frame, "clientHeight", { value: 200 });

        await pointer(frame, "pointerdown", 100);
        await pointer(frame, "pointermove", 140);
        expect(w.emitted("update:position")).toBeUndefined();

        await w.find("button").trigger("click");
        expect(w.text()).toContain("Done");

        await pointer(frame, "pointerdown", 100);
        await pointer(frame, "pointermove", 140);
        await pointer(frame, "pointerup", 140);
        await pointer(frame, "pointermove", 200);

        expect(w.emitted("update:position")).toEqual([[30]]);
    });
});

describe("FormCoverImage with the keyboard", () => {
    let w;

    afterEach(() => {
        w?.unmount();
    });

    it("moves the cover with the arrows while repositioning and finishes on Enter or Escape", async () => {
        w = mount(FormCoverImage, {
            props: {
                src: "/c.png",
                position: 50,
                repositionable: true,
                "onUpdate:position": (position) => w.setProps({ position }),
            },
            attachTo: document.body,
        });
        const frame = w.find("div");

        await frame.trigger("keydown", { key: "ArrowUp" });
        expect(w.emitted("update:position")).toBeUndefined();
        expect(frame.attributes("tabindex")).toBeUndefined();

        await w.find("button").trigger("click");
        await nextTick();

        expect(document.activeElement).toBe(frame.element);
        expect(frame.attributes("role")).toBe("slider");
        expect(frame.attributes("aria-label")).toBe("Cover image position");
        expect(document.getElementById(frame.attributes("aria-describedby")).textContent).toContain(
            "arrow keys",
        );

        await frame.trigger("keydown", { key: "ArrowUp" });
        await frame.trigger("keydown", { key: "ArrowDown" });
        await frame.trigger("keydown", { key: "ArrowDown" });

        expect(w.emitted("update:position")).toEqual([[55], [50], [45]]);
        expect(frame.attributes("aria-valuenow")).toBe("45");

        for (const key of ["Enter", "Escape"]) {
            await frame.trigger("keydown", { key });
            await nextTick();

            expect(frame.attributes("role")).toBeUndefined();
            expect(document.activeElement).toBe(w.find("button").element);

            await w.find("button").trigger("click");
            await nextTick();
        }
    });
});

describe("FormViewSettings images", () => {
    const form = { id: "", workspace_id: "w1", table_id: "t1", cover_image: "", logo: "" };

    it("hands a picked image to the builder and shows the one it is given", async () => {
        const w = mount(FormViewSettings, { props: { form, coverSrc: "blob:cover" } });
        const file = new File(["png"], "a.png", { type: "image/png" });

        await pickFile(w.find("input[aria-label='Cover image']"), file);

        expect(w.emitted("pick-image")).toEqual([["cover", file]]);
        expect(w.find("img[alt='Form cover image']").attributes("src")).toBe("blob:cover");
    });

    it("lets a new form pick its images and asks for the removal of a shown one", async () => {
        const w = mount(FormViewSettings, { props: { form, logoSrc: "blob:logo" } });

        expect(
            w
                .findAll("button")
                .filter((b) => b.text() === "Upload" || b.text() === "Replace")
                .every((b) => b.attributes("disabled") === undefined),
        ).toBe(true);

        await w
            .findAll("button")
            .find((b) => b.text() === "Remove")
            .trigger("click");

        expect(w.emitted("remove-image")).toEqual([["logo"]]);
    });
});
