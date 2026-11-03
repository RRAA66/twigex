// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { conditionNeedsValue, formFieldType } from "@/utils/projects/forms";

const WIDGET_BY_TYPE = { date: "date", assignee: "assignee", checkbox: "checkbox" };

// The input a condition's value is picked with. Contains matches part of
// the text, so it always takes text.
export function conditionValueWidget(field, operator) {
    if (!conditionNeedsValue(operator)) return "none";
    if (operator === "contains") return "text";
    if (field.field_type === "status") return "status";
    if (field.linked_id || field.parent_table_id) return "linked";

    return WIDGET_BY_TYPE[formFieldType(field)] || "text";
}
