// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { t } from "@/i18n/index.js";
import { fieldLabel } from "@/utils/projects/rows";

const DEFAULT_ACCENT_COLOR = "#4f46e5";

const DEFAULT_FIELD_NAMES = {
    name: "projects.grid_view.name",
    status: "projects.grid_view.status",
    assignee: "projects.grid_view.assignee",
    description: "projects.grid_view.description",
    start_date: "projects.grid_view.start_date",
    due_date: "projects.grid_view.due_date",
};

const SKIPPED_FIELDS = new Set(["id", "created_by", "created_at", "updated_at", "deleted_at"]);

const NEW_FIELD_SORT_OFFSET = 9000;

export function headerLabel(header) {
    const key = DEFAULT_FIELD_NAMES[header?.name];

    return key ? t.value(key) : fieldLabel(header);
}

export function emptyFormData(workspaceId, tableId) {
    return normalizeFormData({}, workspaceId, tableId);
}

export function normalizeFormData(data, workspaceId, tableId) {
    return {
        id: data.id || "",
        workspace_id: data.workspace_id || workspaceId,
        table_id: data.table_id || tableId,
        name: data.name || "",
        title: data.title || "",
        description: data.description || "",
        cover_image: data.cover_image || "",
        cover_position: data.cover_position ?? 50,
        logo: data.logo || "",
        accent_color: data.accent_color || DEFAULT_ACCENT_COLOR,
        public_token: data.public_token || "",
        requires_login: data.requires_login || false,
        active: data.active ?? true,
        created_by: data.created_by || "",
    };
}

// Every field of the table is listed: a saved field keeps its settings, a
// field added to the table since the form was saved comes last and hidden,
// and the name is always shown and required because a row needs one.
export function mergeFormFields(headers, savedFields) {
    const savedByName = new Map(savedFields.map((f) => [f.header_name, f]));

    const merged = headers
        .filter(
            (h) =>
                !SKIPPED_FIELDS.has(h.name) &&
                (h.header_usage || "").toLowerCase() !== "calculations",
        )
        .map((h, i) => {
            const saved = savedByName.get(h.name);

            if (saved) return { ...saved, column_position: saved.column_position || "full" };

            return {
                id: "",
                header_id: h.id || h.name,
                header_name: h.name,
                label: headerLabel(h),
                placeholder: "",
                description: "",
                required: false,
                visible: false,
                sort_order: NEW_FIELD_SORT_OFFSET + i,
                column_position: "full",
            };
        });

    merged.sort((a, b) => a.sort_order - b.sort_order);

    const nameField = merged.find((f) => f.header_name === "name");

    if (nameField) {
        nameField.required = true;
        nameField.visible = true;
    }

    return merged;
}
