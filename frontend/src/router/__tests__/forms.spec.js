// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

vi.mock("@/utils/utils", async (importOriginal) => {
    const original = await importOriginal();

    return { ...original, hasSessionCookie: () => true };
});

const config = vi.fn();

vi.mock("@/services/configService", () => ({ default: { config: () => config() } }));

import router from "@/router";
import { useSettingsStore } from "@/store/settings";

const formPages = [
    { name: "form-view-create", params: { id: "w1", tid: "t1" } },
    { name: "form-view", params: { id: "w1", tid: "t1", fid: "f1" } },
];

describe("form pages", () => {
    beforeEach(async () => {
        setActivePinia(createPinia());
        config.mockReset();
        await router.push("/profile");
    });

    for (const page of formPages) {
        it(`${page.name} does not open with Forms off`, async () => {
            useSettingsStore().setConfig({ FeatureFlags: { Forms: false } });

            await router.push(page);

            expect(router.currentRoute.value.name).not.toBe(page.name);
        });

        it(`${page.name} opens with Forms on`, async () => {
            useSettingsStore().setConfig({ FeatureFlags: { Forms: true } });

            await router.push(page);

            expect(router.currentRoute.value.name).toBe(page.name);
        });
    }

    it("loads the signed-in config when it only has the signed-out one", async () => {
        useSettingsStore().setConfig({ DefaultLocale: "en" });
        config.mockResolvedValue({ data: { FeatureFlags: { Forms: true } } });

        await router.push(formPages[1]);

        expect(config).toHaveBeenCalled();
        expect(router.currentRoute.value.name).toBe("form-view");
    });
});
