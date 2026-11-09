// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"context"
	"testing"
)

func TestUpdateTableIconShowsInTheTableList(t *testing.T) {
	requireDB(t)
	cleanTables(t, "workspace_tables", "workspace_members")
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	mustExec(t, `INSERT INTO workspace_members (id, user_id, workspace_id, role, date_joined) VALUES ('m1', 'u1', 'w1', 'member', 1)`)
	mustExec(t, `INSERT INTO workspace_tables (id, workspace_id, name, display_name, linked, single_select, parent_table_id, both_direction_link, second_table_id, folder_id, deleted_at)
		VALUES ('t1', 'w1', 'zz_icon', '', false, false, '', false, '', '', 0)`)

	if err := w.UpdateTableIcon(ctx, "w2", "t1", "StarIcon"); err != nil {
		t.Fatal(err)
	}

	if err := w.UpdateTableIcon(ctx, "w1", "t1", "FlagIcon"); err != nil {
		t.Fatal(err)
	}

	tables, err := w.GetTableList(ctx, "w1", "u1")
	if err != nil {
		t.Fatal(err)
	}

	if len(tables) != 1 || tables[0].Icon != "FlagIcon" {
		t.Errorf("got %+v, want t1 with FlagIcon; another workspace's id must not change it", tables)
	}
}
