// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import FormFileInput from "@/components/Projects/Form/FormFileInput.vue";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";

vi.mock("@/services/formService", () => ({
    default: { uploadPublicFormFiles: vi.fn() },
}));

const uploaded = { id: "file1", name: "a.pdf", size: 3, type: "application/pdf" };

function mountInput(modelValue = null) {
    return mount(FormFileInput, { props: { id: "docs", token: "tok", modelValue } });
}

function pickFiles(input, files) {
    Object.defineProperty(input.element, "files", { value: files, configurable: true });

    return input.trigger("change");
}

describe("FormFileInput", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        setActivePinia(createPinia());
    });

    it("adds the uploaded files to those already there", async () => {
        formService.uploadPublicFormFiles.mockResolvedValue({ data: [uploaded] });
        const earlier = { id: "file0", name: "old.png" };
        const w = mountInput([earlier]);

        await pickFiles(w.find("input[type='file']"), [new File(["pdf"], "a.pdf")]);
        await flushPromises();

        expect(w.emitted("update:modelValue")).toEqual([[[earlier, uploaded]]]);
    });

    it("keeps the files on the screen without a link to upload through", async () => {
        const w = mount(FormFileInput, { props: { id: "docs", modelValue: null } });

        await pickFiles(w.find("input[type='file']"), [new File(["pdf"], "a.pdf")]);
        await flushPromises();

        expect(formService.uploadPublicFormFiles).not.toHaveBeenCalled();
        expect(w.emitted("update:modelValue")).toEqual([
            [[expect.objectContaining({ name: "a.pdf", size: 3 })]],
        ]);
    });

    it("does not send the form it sits in when files are added", async () => {
        const onSubmit = vi.fn((event) => event.preventDefault());
        const w = mount(
            {
                components: { FormFileInput },
                template: `<form @submit="onSubmit"><FormFileInput id="docs" token="tok" /></form>`,
                setup: () => ({ onSubmit }),
            },
            { attachTo: document.body },
        );

        await w.find("button").trigger("click");

        expect(onSubmit).not.toHaveBeenCalled();
        w.unmount();
    });

    it("takes a file back off the field", async () => {
        const w = mountInput([uploaded, { id: "file2", name: "b.pdf" }]);

        await w.find("button[aria-label='Remove a.pdf']").trigger("click");

        expect(w.emitted("update:modelValue")).toEqual([[[{ id: "file2", name: "b.pdf" }]]]);
    });

    it("alerts when the upload is refused and keeps the field as it was", async () => {
        formService.uploadPublicFormFiles.mockRejectedValue({
            response: { status: 400, data: { error: "Files of this type cannot be uploaded." } },
        });
        const w = mountInput();
        const showError = vi.spyOn(useAlertStore(), "showError");

        await pickFiles(w.find("input[type='file']"), [new File(["MZ"], "run.exe")]);
        await flushPromises();

        expect(showError).toHaveBeenCalledWith("Files of this type cannot be uploaded.");
        expect(w.emitted("update:modelValue")).toBeUndefined();
    });

    it("sends no more files than a field keeps and says so", async () => {
        formService.uploadPublicFormFiles.mockResolvedValue({ data: [uploaded] });
        const nine = Array.from({ length: 9 }, (_, i) => ({ id: `f${i}`, name: `${i}.pdf` }));
        const w = mountInput(nine);
        const showError = vi.spyOn(useAlertStore(), "showError");

        await pickFiles(w.find("input[type='file']"), [
            new File(["1"], "x.pdf"),
            new File(["2"], "y.pdf"),
        ]);
        await flushPromises();

        expect(formService.uploadPublicFormFiles.mock.calls[0][1]).toHaveLength(1);
        expect(showError).toHaveBeenCalledWith(
            "A field takes up to 10 files. The others were left out.",
        );
    });

    it("says nothing when every file fits", async () => {
        formService.uploadPublicFormFiles.mockResolvedValue({ data: [uploaded] });
        const w = mountInput();
        const showError = vi.spyOn(useAlertStore(), "showError");

        await pickFiles(w.find("input[type='file']"), [new File(["1"], "x.pdf")]);
        await flushPromises();

        expect(showError).not.toHaveBeenCalled();
    });
});
