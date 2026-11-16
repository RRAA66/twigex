// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/twigex/twigex/interfaces"
	"github.com/twigex/twigex/model"
)

type fakeCreateViewStore struct {
	fakeReadAccessStore
	names   map[string]bool
	created []string
}

func (f *fakeCreateViewStore) ViewExistsByName(_, name string) (bool, error) {
	return f.names[name], nil
}

func (f *fakeCreateViewStore) GetCustomFields(_, _ string) ([]model.TaskOrderField, error) {
	return []model.TaskOrderField{{Name: "budget", DisplayName: "Budget"}}, nil
}

func (f *fakeCreateViewStore) CreateView(workspaceID, tableID, order, name, viewType, _, _, viewID string) (*model.WorkspaceView, error) {
	f.created = append(f.created, order)
	return &model.WorkspaceView{ID: viewID, WorkspaceID: workspaceID, TableID: tableID, Name: name, ViewType: viewType, TaskOrder: order}, nil
}

// tableViewRoles answers like the enterprise roles: a member's workspace and
// per-table roles add up.
type tableViewRoles struct{}

func (tableViewRoles) Resolve(in interfaces.AccessInput) interfaces.WorkspaceAccess {
	return tableViewAccess{in: in}
}

type tableViewAccess struct {
	openAccess
	in interfaces.AccessInput
}

func (v tableViewAccess) CanPerformRowAction(tableID, workspacePerm, tableAction string) bool {
	for _, r := range v.in.Roles {
		if !r.PerTableMode && slices.Contains(r.Permissions, workspacePerm) {
			return true
		}
	}

	return slices.Contains(v.in.TablePerms, model.TablePermission{TableID: tableID, Action: tableAction})
}

func createViewApp(memberRole string) (*App, *fakeCreateViewStore) {
	return createViewAppWithRole(model.ProjectWorkspaceRole{
		ID: memberRole + "-id", Name: memberRole, Permissions: []string{model.PermissionCreateWorkspaceView},
	})
}

func createViewAppWithRole(role model.ProjectWorkspaceRole, tablePerms ...model.TablePermission) (*App, *fakeCreateViewStore) {
	return createViewAppWithRoles([]model.ProjectWorkspaceRole{role}, tablePerms...)
}

func createViewAppWithRoles(roles []model.ProjectWorkspaceRole, tablePerms ...model.TablePermission) (*App, *fakeCreateViewStore) {
	var names []string
	for _, r := range roles {
		names = append(names, r.Name)
	}

	ws := &fakeCreateViewStore{
		fakeReadAccessStore: fakeReadAccessStore{
			fakeAccessWorkspaceStore: fakeAccessWorkspaceStore{
				memberRoleName: strings.Join(names, " "),
				roles:          roles,
				tablePerms:     tablePerms,
			},
			tableOwner: map[string]string{"t-a": "ws-a"},
		},
		names: map[string]bool{"Taken": true},
	}
	a := readAccessApp(&ws.fakeReadAccessStore)
	a.Store.Workspace = ws
	a.WorkspaceRoles = tableViewRoles{}
	return a, ws
}

func TestCreateGridViewStartsWithTheTableColumns(t *testing.T) {
	a, ws := createViewApp("member")
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	view, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Mine", "grid", true, member, "", "")
	if appErr != nil {
		t.Fatalf("CreateView: %v", appErr)
	}

	var columns []model.TaskOrderField
	if err := json.Unmarshal([]byte(view.TaskOrder), &columns); err != nil {
		t.Fatalf("grid order %q is not a column list: %v", view.TaskOrder, err)
	}
	has := map[string]bool{}
	for _, c := range columns {
		has[c.Name] = true
	}
	for _, name := range []string{"name", "status", "budget"} {
		if !has[name] {
			t.Errorf("grid columns %v lack %q", columns, name)
		}
	}
	if len(ws.created) != 1 {
		t.Errorf("created %d views, want 1", len(ws.created))
	}
}

func TestCreateGridViewRefusesATakenName(t *testing.T) {
	a, ws := createViewApp("member")
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	_, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Taken", "grid", true, member, "", "")
	if appErr == nil || appErr.Status != http.StatusConflict {
		t.Errorf("a grid view named like another: got %v, want 409", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d views, want none", len(ws.created))
	}
}

func TestCreateViewNeedsTheWorkspacePermissionOutsidePerTableMode(t *testing.T) {
	a, ws := createViewAppWithRole(model.ProjectWorkspaceRole{ID: "reader-id", Name: "reader", Permissions: []string{model.PermissionCreateTask}})
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	_, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Mine", "grid", true, member, "", "")
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("got %v, want 403", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d views, want none", len(ws.created))
	}
}

func TestCreateViewIsOpenToAPerTableRoleThatManagesTheTablesViews(t *testing.T) {
	role := model.ProjectWorkspaceRole{ID: "table-id", Name: "table", PerTableMode: true}
	a, ws := createViewAppWithRole(role, model.TablePermission{TableID: "t-a", Action: "manage_views"})
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	if _, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Mine", "grid", true, member, "", ""); appErr != nil {
		t.Fatalf("CreateView: %v", appErr)
	}
	if len(ws.created) != 1 {
		t.Errorf("created %d views, want 1", len(ws.created))
	}
}

func TestCreateViewAddsAPerTableRoleThatManagesTheTablesViewsToAWorkspaceRole(t *testing.T) {
	reader := model.ProjectWorkspaceRole{ID: "reader-id", Name: "reader", Permissions: []string{model.PermissionCreateTask}}
	table := model.ProjectWorkspaceRole{ID: "table-id", Name: "table", PerTableMode: true}
	a, ws := createViewAppWithRoles([]model.ProjectWorkspaceRole{reader, table}, model.TablePermission{TableID: "t-a", Action: "manage_views"})
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	if _, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Mine", "grid", true, member, "", ""); appErr != nil {
		t.Fatalf("CreateView: %v", appErr)
	}
	if len(ws.created) != 1 {
		t.Errorf("created %d views, want 1", len(ws.created))
	}
}

func TestCreateViewRefusesAPerTableRoleThatDoesNotManageTheTablesViews(t *testing.T) {
	role := model.ProjectWorkspaceRole{ID: "table-id", Name: "table", PerTableMode: true}
	a, ws := createViewAppWithRole(role, model.TablePermission{TableID: "t-other", Action: "manage_views"})
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	_, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Mine", "grid", true, member, "", "")
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("got %v, want 403", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d views, want none", len(ws.created))
	}
}

func TestDeleteViewNeedsTheWorkspacePermissionOutsidePerTableMode(t *testing.T) {
	a, _ := createViewAppWithRole(model.ProjectWorkspaceRole{ID: "reader-id", Name: "reader", Permissions: []string{model.PermissionCreateTask}})
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	_, appErr := a.DeleteWorkspaceTableView(context.Background(), "ws-a", "t-a", "view-1", member)
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("got %v, want 403", appErr)
	}
}

func TestCreateViewRefusesAFormView(t *testing.T) {
	a, ws := createViewApp("member")
	member := model.User{ID: "u1", Role: model.SystemUserRoleId}

	_, appErr := a.CreateView(context.Background(), "ws-a", "t-a", "", "Signup", "form", true, member, "", "")
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Errorf("got %v, want 400", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d views, want none", len(ws.created))
	}
}
