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
    });

    it("keeps an image picked again while the one before it was being sent", async () => {
        const sent = deferred();

        formService.setFormImage.mockReturnValueOnce(sent.promise);
        const { images } = setup();
        const first = new File(["a"], "a.png");
        const second = new File(["b"], "b.png");
        const stored = vi.fn();

        images.pick("cover", first);
        const saving = images.save(saved, stored);

        images.pick("cover", second);
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

        images.pick("cover", new File(["a"], "a.png"));
        images.pick("logo", new File(["l"], "l.png"));
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
    ])("refuses a file with %s as soon as it is picked", (_, file) => {
        const { images } = setup();
        const showError = vi.spyOn(useAlertStore(), "showError");

        images.pick("cover", file);

        expect(showError).toHaveBeenCalledWith("Use a JPEG, PNG, GIF or WebP image.");
        expect(images.src("cover")).toBe("");
    });

    it("takes an image the browser gives no type, by its extension", () => {
        const { images } = setup();

        images.pick("logo", new File(["x"], "Logo.WEBP"));

        expect(images.src("logo")).toBe("blob:Logo.WEBP");
    });

    it("refuses an image over 5 MB", () => {
        const { images } = setup();
        const showError = vi.spyOn(useAlertStore(), "showError");
        const big = new File(["x"], "big.png");

        Object.defineProperty(big, "size", { value: (5 << 20) + 1 });
        images.pick("cover", big);

        expect(showError).toHaveBeenCalledWith("Use an image of 5 MB or less.");
    });
});
