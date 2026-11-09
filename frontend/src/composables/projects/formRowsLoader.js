// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import workspaceService from "@/services/workspaceService";

// The builder has no form link to list rows through, so it asks the workspace.
// The rows are not counted, so one row past the page tells there are more.
export function formRowsLoader(workspaceId, tableId) {
    return async ({ page, limit, search }) => {
        const { data } = await workspaceService.getLinkedRecordsLite(workspaceId, tableId, {
            q: search,
            limit: limit + 1,
            offset: (page - 1) * limit,
        });
        const rows = (data || []).map((row) => ({
            id: String(row.id),
            name: row.name,
            color: row.color || undefined,
        }));

        return { items: rows.slice(0, limit), has_more: rows.length > limit };
    };
}
