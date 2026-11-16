// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"os"
	"testing"
)

func TestFormViewPermissionMigrationKeepsARolesOtherPermissions(t *testing.T) {
	requireDB(t)
	t.Cleanup(func() { _, _ = testDB.Exec("DELETE FROM workspace_roles WHERE workspace_id = 'zz-mig100'") })
	mustExec(t, `INSERT INTO workspace_roles (id, workspace_id, name, permissions) VALUES
		('zz-user', 'zz-mig100', 'user', 'create_task update_task views_grid'),
		('zz-admin', 'zz-mig100', 'admin', 'create_task views_form delete_task'),
		('zz-empty', 'zz-mig100', 'user', NULL),
		('zz-custom', 'zz-mig100', 'sales', 'create_task')`)

	sql, err := os.ReadFile("../../db/mysql/101_add_form_view_permission.up.sql")
	if err != nil {
		t.Fatal(err)
	}

	mustExec(t, string(sql))

	want := map[string]string{
		"zz-user":   "create_task update_task views_grid views_form",
		"zz-admin":  "create_task views_form delete_task",
		"zz-empty":  "views_form",
		"zz-custom": "create_task",
	}
	for id, permissions := range want {
		var got string
		if err := testDB.QueryRow("SELECT COALESCE(permissions, '') FROM workspace_roles WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatal(err)
		}

		if got != permissions {
			t.Errorf("%s: permissions %q, want %q", id, got, permissions)
		}
	}
}
