// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, it } from "vitest";
import { conditionValueWidget } from "../conditionValueWidget";

describe("conditionValueWidget", () => {
    it.each([
        [{ field_type: "status", linked_id: "t-status", single_select: true }, "status"],
        [{ field_type: "link", linked_id: "t-link", parent_table_id: "t-clients" }, "linked"],
        [{ field_type: "single select", linked_id: "t-options", single_select: true }, "linked"],
        [{ field_type: "", linked_id: "t-link" }, "linked"],
        [{ field_type: "default_assignee" }, "assignee"],
        [{ field_type: "date" }, "date"],
        [{ field_type: "default_date" }, "date"],
        [{ field_type: "bool" }, "checkbox"],
        [{ field_type: "number" }, "text"],
        [{ field_type: "text" }, "text"],
    ])("picks a value for %o with %s", (field, widget) => {
        expect(conditionValueWidget(field, "equals")).toBe(widget);
        expect(conditionValueWidget(field, "not_equals")).toBe(widget);
    });

    it("takes text for contains and nothing for the emptiness checks", () => {
        const field = { field_type: "link", linked_id: "t-link" };

        expect(conditionValueWidget(field, "contains")).toBe("text");
        expect(conditionValueWidget(field, "is_empty")).toBe("none");
        expect(conditionValueWidget(field, "is_not_empty")).toBe("none");
    });
});
