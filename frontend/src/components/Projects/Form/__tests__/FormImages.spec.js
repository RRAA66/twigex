// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick, reactive } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormCoverImage from "@/components/Projects/Form/FormCoverImage.vue";
import FormViewSettings from "@/components/Projects/Form/FormViewSettings.vue";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";

vi.mock("@/services/formService", () => ({
    default: { setFormImage: vi.fn(), removeFormImage: vi.fn() },
}));

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
    let form;

    // Applies each update as the builder does, to a new form object.
    function mountSettings() {
        const w = mount(FormViewSettings, {
            props: {
                form,
                onUpdate: (patch) => w.setProps({ form: { ...w.props("form"), ...patch } }),
            },
        });

        return w;
    }

    beforeEach(() => {
        vi.clearAllMocks();
        setActivePinia(createPinia());
        form = reactive({
            id: "f1",
            workspace_id: "w1",
            table_id: "t1",
            cover_image: "",
            logo: "",
        });
    });

    it("uploads a cover and shows it", async () => {
        formService.setFormImage.mockResolvedValue({ data: { cover_image: "c1", logo: "" } });
        const w = mountSettings();
        const file = new File(["png"], "a.png", { type: "image/png" });

        await pickFile(w.find("input[aria-label='Cover image']"), file);
        await flushPromises();

        expect(formService.setFormImage).toHaveBeenCalledWith("w1", "t1", "f1", "cover", file);
        expect(w.emitted("update")).toEqual([[{ cover_image: "c1" }]]);
        expect(w.find("img[alt='Form cover image']").attributes("src")).toBe(
            "/api/workspaces/w1/file/c1",
        );
    });

    it("removes the logo", async () => {
        form.logo = "l1";
        formService.removeFormImage.mockResolvedValue({ data: { cover_image: "", logo: "" } });
        const w = mountSettings();

        const remove = w.findAll("button").find((b) => b.text() === "Remove");

        await remove.trigger("click");
        await flushPromises();

        expect(formService.removeFormImage).toHaveBeenCalledWith("w1", "t1", "f1", "logo");
        expect(w.emitted("update")).toEqual([[{ logo: "" }]]);
    });

    it("drops the stored image when another form was opened meanwhile", async () => {
        let answer;

        formService.setFormImage.mockReturnValue(
            new Promise((resolve) => {
                answer = resolve;
            }),
        );
        const w = mountSettings();

        await pickFile(w.find("input[aria-label='Cover image']"), new File(["png"], "a.png"));
        await w.setProps({ form: { ...form, id: "f2", cover_image: "" } });
        answer({ data: { cover_image: "c1", logo: "" } });
        await flushPromises();

        expect(w.emitted("update")).toBeUndefined();
        expect(w.props("form").cover_image).toBe("");
    });

    it("shows the server's refusal and keeps the old image", async () => {
        form.cover_image = "c0";
        formService.setFormImage.mockRejectedValue({
            response: { status: 400, data: { error: "Use a JPEG, PNG, GIF or WebP image." } },
        });
        const w = mountSettings();
        const alertStore = useAlertStore();
        const showError = vi.spyOn(alertStore, "showError");

        await pickFile(w.find("input[aria-label='Cover image']"), new File(["x"], "a.svg"));
        await flushPromises();

        expect(showError).toHaveBeenCalledWith("Use a JPEG, PNG, GIF or WebP image.");
        expect(w.emitted("update")).toBeUndefined();
    });

    it("asks for the form to be saved before images are added", () => {
        form.id = "";
        const w = mountSettings();

        expect(w.text()).toContain("Save the form to add a cover image and a logo.");
        expect(
            w
                .findAll("button")
                .filter((b) => b.text() === "Upload")
                .every((b) => b.attributes("disabled") !== undefined),
        ).toBe(true);
    });
});
