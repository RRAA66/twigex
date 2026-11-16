// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, ref } from "vue";
import { flushPromises } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { useFormImages } from "../useFormImages";

vi.mock("@/services/formService", () => ({
    default: { setFormImage: vi.fn(), removeFormImage: vi.fn() },
}));

const saved = { id: "f1", workspace_id: "w1", table_id: "t1", cover_image: "", logo: "" };

function deferred() {
    let resolve;
    const promise = new Promise((r) => (resolve = r));

    return { promise, resolve };
}

function setup() {
    const form = ref({ ...saved });
    const images = effectScope().run(() => useFormImages(form));

    return { form, images };
}

describe("useFormImages", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        setActivePinia(createPinia());
        vi.spyOn(URL, "createObjectURL").mockImplementation((file) => `blob:${file.name}`);
        vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
        // jsdom loads no images: a file named "broken…" fails to load as one.
        vi.stubGlobal(
            "Image",
            class {
                set src(url) {
                    queueMicrotask(() => (url.includes("broken") ? this.onerror() : this.onload()));
                }
            },
        );
    });

    it("keeps an image picked again while the one before it was being sent", async () => {
        const sent = deferred();

        formService.setFormImage.mockReturnValueOnce(sent.promise);
        const { images } = setup();
        const first = new File(["a"], "a.png");
        const second = new File(["b"], "b.png");
        const stored = vi.fn();

        await images.pick("cover", first);
        const saving = images.save(saved, stored);

        await images.pick("cover", second);
        sent.resolve({ data: { cover_image: "c1" } });
        await saving;

        expect(stored).toHaveBeenCalledWith({ cover_image: "c1" });
        expect(images.src("cover")).toBe("blob:b.png");
        expect(URL.revokeObjectURL).not.toHaveBeenCalledWith("blob:b.png");

        formService.setFormImage.mockResolvedValue({ data: { cover_image: "c2" } });
        await images.save(saved, stored);

        expect(formService.setFormImage).toHaveBeenLastCalledWith(
            "w1",
            "t1",
            "f1",
            "cover",
            second,
        );
    });

    it("gives a form opened meanwhile nothing of the one that was saved", async () => {
        const sent = deferred();

        formService.setFormImage.mockReturnValueOnce(sent.promise);
        const { form, images } = setup();
        const stored = vi.fn();

        await images.pick("cover", new File(["a"], "a.png"));
        await images.pick("logo", new File(["l"], "l.png"));
        const saving = images.save(saved, stored);

        form.value = { ...saved, id: "f2" };
        images.discard();
        sent.resolve({ data: { cover_image: "c1" } });
        await saving;
        await flushPromises();

        expect(stored).not.toHaveBeenCalled();
        expect(formService.setFormImage).toHaveBeenCalledTimes(1);
    });

    it.each([
        ["a type the server refuses", new File(["x"], "a.svg", { type: "image/png" })],
        ["no extension", new File(["x"], "cover", { type: "image/png" })],
    ])("refuses a file with %s as soon as it is picked", async (_, file) => {
        const { images } = setup();
        const showError = vi.spyOn(useAlertStore(), "showError");

        await images.pick("cover", file);

        expect(showError).toHaveBeenCalledWith("Use a JPEG, PNG, GIF or WebP image.");
        expect(images.src("cover")).toBe("");
    });

    it("says whether an image waits to be saved", async () => {
        formService.setFormImage.mockResolvedValue({ data: { logo: "img1" } });
        const { images } = setup();

        expect(images.hasPending.value).toBe(false);

        await images.pick("logo", new File(["x"], "logo.png"));

        expect(images.hasPending.value).toBe(true);

        await images.save(saved, () => {});

        expect(images.hasPending.value).toBe(false);
    });

    it("takes an image the browser gives no type, by its extension", async () => {
        const { images } = setup();

        await images.pick("logo", new File(["x"], "Logo.WEBP"));

        expect(images.src("logo")).toBe("blob:Logo.WEBP");
    });

    it("refuses a file that does not load as an image and keeps the image before it", async () => {
        const { images } = setup();
        const showError = vi.spyOn(useAlertStore(), "showError");

        await images.pick("cover", new File(["a"], "a.png"));
        await images.pick("cover", new File(["x"], "broken.png"));

        expect(showError).toHaveBeenCalledWith("This file is not an image that can be shown.");
        expect(images.src("cover")).toBe("blob:a.png");
        expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:broken.png");
        expect(URL.revokeObjectURL).not.toHaveBeenCalledWith("blob:a.png");
    });

    it("leaves no unsaved change for a file that does not load as an image", async () => {
        const { images } = setup();

        await images.pick("logo", new File(["x"], "broken.png"));

        expect(images.hasPending.value).toBe(false);
        expect(images.src("logo")).toBe("");
    });

    it("keeps the image picked last when an earlier one loads after it", async () => {
        const { images } = setup();

        const first = images.pick("cover", new File(["a"], "a.png"));

        await images.pick("cover", new File(["b"], "b.png"));
        await first;

        expect(images.src("cover")).toBe("blob:b.png");
        expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:a.png");
    });

    it("refuses an image over 5 MB", async () => {
        const { images } = setup();
        const showError = vi.spyOn(useAlertStore(), "showError");
        const big = new File(["x"], "big.png");

        Object.defineProperty(big, "size", { value: (5 << 20) + 1 });
        await images.pick("cover", big);

        expect(showError).toHaveBeenCalledWith("Use an image of 5 MB or less.");
    });
});
