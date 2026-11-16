// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { currentLocale } from "@/i18n/index.js";
import userService from "@/services/userService";
import PublicFormView from "@/views/Public/PublicFormView.vue";

vi.mock("@/services/userService", () => ({
    default: { preferences: vi.fn() },
}));
vi.mock("vue-router", () => ({
    useRoute: () => ({ params: { token: "tok" } }),
}));

async function mountView() {
    const pinia = createPinia();

    setActivePinia(pinia);

    const w = mount(PublicFormView, {
        global: { plugins: [pinia], stubs: { FormFill: true } },
    });

    await flushPromises();

    return w;
}

describe("PublicFormView", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        currentLocale.value = "en";
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        currentLocale.value = "en";
    });

    it("shows the form in the language the user chose", async () => {
        userService.preferences.mockResolvedValue({
            data: { DisplaySettings: [{ name: "language", value: "lv" }] },
        });

        const w = await mountView();

        await vi.waitFor(() => expect(currentLocale.value).toBe("lv"));
        w.unmount();
    });

    it("falls back to the browser's language without the user's choice", async () => {
        userService.preferences.mockRejectedValue({ response: { status: 401 } });
        vi.stubGlobal("navigator", { language: "pl-PL" });

        const w = await mountView();

        await vi.waitFor(() => expect(currentLocale.value).toBe("pl"));
        w.unmount();
    });

    it("keeps the default for a browser language the app does not have", async () => {
        userService.preferences.mockResolvedValue({ data: {} });
        vi.stubGlobal("navigator", { language: "de-DE" });

        const w = await mountView();

        expect(currentLocale.value).toBe("en");
        w.unmount();
    });
});
