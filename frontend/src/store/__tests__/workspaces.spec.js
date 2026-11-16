// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { beforeEach, describe, expect, it } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useWorkspaceStore } from "../workspaces";

describe("getTableIcon", () => {
    let store;

    beforeEach(() => {
        setActivePinia(createPinia());
        store = useWorkspaceStore();
    });

    it("follows an icon change to a table in the loaded tree", () => {
        store.workspaceFolders = [{ id: "f1", children: [{ id: "t1", icon: "StarIcon" }] }];

        expect(store.getTableIcon("w1", "t1")).toBe("StarIcon");

        store.setTableIcon("w1", "t1", "BoltIcon");

        expect(store.getTableIcon("w1", "t1")).toBe("BoltIcon");
    });

    it("knows nothing of a table no loaded tree holds", () => {
        expect(store.getTableIcon("w1", "t9")).toBeNull();
    });
});
