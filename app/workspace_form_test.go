// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twigex/twigex/config"
	"github.com/twigex/twigex/interfaces"
	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

type fakeFormStore struct {
	fakeAccessWorkspaceStore
	forms  map[string]*model.WorkspaceFormWithFields
	writes int
}

func (f *fakeFormStore) GetTableWorkspaceID(_ context.Context, tableID string) (string, error) {
	if tableID == tbl1 || tableID == tbl2 {
		return wsID, nil
	}

	return "", nil
}

func (f *fakeFormStore) GetTableName(tableID string) (string, error) {
	return "tbl_" + tableID, nil
}

func (f *fakeFormStore) GetColumnNames(_ context.Context, _ string) ([]string, error) {
	return []string{"id", "name", "status", "budget", "created_at", "parent_task_id"}, nil
}

func (f *fakeFormStore) ViewExistsByName(tableID, name string) (bool, error) {
	for _, form := range f.forms {
		if form.TableID == tableID && form.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func (f *fakeFormStore) CreateForm(_ context.Context, form model.WorkspaceForm, fields []model.WorkspaceFormField) error {
	f.writes++
	f.forms[form.ID] = &model.WorkspaceFormWithFields{WorkspaceForm: form, Fields: fields}
	return nil
}

func (f *fakeFormStore) GetForm(_ context.Context, formID string) (*model.WorkspaceFormWithFields, error) {
	form, ok := f.forms[formID]
	if !ok {
		return nil, nil
	}

	stored := *form
	return &stored, nil
}

func (f *fakeFormStore) UpdateForm(_ context.Context, form model.WorkspaceForm, fields []model.WorkspaceFormField) error {
	f.writes++
	stored := f.forms[form.ID]
	stored.Name = form.Name
	stored.Title = form.Title
	stored.UpdatedAt = form.UpdatedAt
	stored.Fields = fields
	return nil
}

func (f *fakeFormStore) DeleteForm(_ context.Context, formID string) error {
	f.writes++
	delete(f.forms, formID)
	return nil
}

func formApp(access fakeAccessWorkspaceStore, licensed bool) (*App, *fakeFormStore) {
	ws := &fakeFormStore{
		fakeAccessWorkspaceStore: access,
		forms: map[string]*model.WorkspaceFormWithFields{
			"form-1": {WorkspaceForm: model.WorkspaceForm{
				ID: "form-1", WorkspaceID: wsID, TableID: tbl1, Name: "Intake",
				PublicToken: "token-1", CreatedBy: "creator", Active: true,
			}},
		},
	}

	a := &App{
		Store: store.Store{Workspace: ws},
		ConfigStore: config.ConfigStore{Config: &model.ServerConfig{
			SqlSettings: model.SqlSettings{QueryTimeout: model.NewInt(25)},
		}},
	}
	if licensed {
		a.Server.License = &model.License{
			StartsAt:  time.Now().Add(-time.Hour).Unix(),
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			Features:  &model.Features{Forms: boolPtr(true)},
		}
	}

	return a, ws
}

// formRoles stands in for enterprise roles that grant no table permissions,
// so only workspace permissions let a member manage views, and that hide one
// table from everyone they resolve.
type formRoles struct{ hiddenTable string }

func (r formRoles) Resolve(interfaces.AccessInput) interfaces.WorkspaceAccess {
	return formAccess(r)
}

type formAccess struct{ hiddenTable string }

func (formAccess) AssignedOnlyAll() bool                   { return false }
func (formAccess) AssignedOnly(string) bool                { return false }
func (formAccess) CanPerformRowAction(_, _, _ string) bool { return true }
func (formAccess) TableHasPermission(_, _ string) bool     { return false }
func (f formAccess) VisibleTables(tables []model.WorkspaceTable) []model.WorkspaceTable {
	var visible []model.WorkspaceTable
	for _, t := range tables {
		if t.ID != f.hiddenTable {
			visible = append(visible, t)
		}
	}

	return visible
}

func memberWithoutViewPermissions() fakeAccessWorkspaceStore {
	role := model.ProjectWorkspaceRole{ID: "reader-id", Name: "reader", Permissions: []string{model.PermissionCreateTask}}
	return fakeAccessWorkspaceStore{memberRoleName: role.Name, roles: []model.ProjectWorkspaceRole{role}}
}

// perTableViewManager is a member whose one role is per-table and manages
// the views of tableID only.
func perTableViewManager(tableID string) fakeAccessWorkspaceStore {
	role := model.ProjectWorkspaceRole{ID: "table-id", Name: "table", PerTableMode: true}
	return fakeAccessWorkspaceStore{
		memberRoleName: role.Name,
		roles:          []model.ProjectWorkspaceRole{role},
		tablePerms:     []model.TablePermission{{RoleID: role.ID, TableID: tableID, Action: "manage_views"}},
	}
}

// tableGrantRoles answers table permissions as the enterprise roles do: from
// the table permissions for per-table roles, and yes to anyone else.
type tableGrantRoles struct{}

func (tableGrantRoles) Resolve(in interfaces.AccessInput) interfaces.WorkspaceAccess {
	return tableGrantAccess{in: in}
}

type tableGrantAccess struct {
	openAccess
	in interfaces.AccessInput
}

func (g tableGrantAccess) TableHasPermission(tableID, action string) bool {
	perTable := slices.ContainsFunc(g.in.Roles, func(r model.ProjectWorkspaceRole) bool { return r.PerTableMode })
	if !perTable {
		return true
	}

	return slices.ContainsFunc(g.in.TablePerms, func(p model.TablePermission) bool {
		return p.TableID == tableID && p.Action == action
	})
}

func TestCreateFormStoresItOnTheURLTable(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	request := model.WorkspaceForm{ID: "chosen", WorkspaceID: "other", TableID: "other", Name: "Signup", CreatedBy: "someone"}
	fields := []model.WorkspaceFormField{{HeaderName: "name", Label: "Name"}}

	created, appErr := a.CreateForm(context.Background(), wsID, tbl1, request, fields, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("CreateForm: %v", appErr)
	}

	stored := ws.forms[created.ID]
	if stored == nil {
		t.Fatalf("form %s was not stored", created.ID)
	}
	if created.ID == "chosen" || stored.WorkspaceID != wsID || stored.TableID != tbl1 || stored.CreatedBy != workspaceAdmin.ID {
		t.Errorf("stored %+v, want a new id in %s/%s created by %s", stored.WorkspaceForm, wsID, tbl1, workspaceAdmin.ID)
	}
	if stored.PublicToken == "" || !stored.Active {
		t.Errorf("stored %+v, want an active form with a token", stored.WorkspaceForm)
	}
	if len(stored.Fields) != 1 || stored.Fields[0].FormID != created.ID || stored.Fields[0].ID == "" {
		t.Errorf("stored fields %+v, want one field of the new form", stored.Fields)
	}
}

func TestCreateFormRefusesANonMember(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	_, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, nil, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestUpdateFormRefusesAMemberWhoCannotChangeViews(t *testing.T) {
	a, ws := formApp(memberWithoutViewPermissions(), true)
	a.WorkspaceRoles = formRoles{}

	_, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestFormsRefuseAMemberWithoutViewPermissionsOutsidePerTableMode(t *testing.T) {
	a, ws := formApp(memberWithoutViewPermissions(), true)
	a.WorkspaceRoles = tableGrantRoles{}
	ctx := context.Background()

	_, createErr := a.CreateForm(ctx, wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, nil, accessUser())
	_, getErr := a.GetForm(ctx, wsID, tbl1, "form-1", accessUser())
	_, updateErr := a.UpdateForm(ctx, wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, accessUser())
	deleteErr := a.DeleteForm(ctx, wsID, tbl1, "form-1", accessUser())

	for name, appErr := range map[string]*model.AppError{"create": createErr, "get": getErr, "update": updateErr, "delete": deleteErr} {
		if appErr == nil || appErr.Status != http.StatusForbidden {
			t.Errorf("%s: want 403, got %v", name, appErr)
		}
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestFormsAreOpenToAPerTableRoleThatManagesTheTablesViews(t *testing.T) {
	a, _ := formApp(perTableViewManager(tbl1), true)
	a.WorkspaceRoles = tableGrantRoles{}

	if _, appErr := a.GetForm(context.Background(), wsID, tbl1, "form-1", accessUser()); appErr != nil {
		t.Fatalf("GetForm: %v", appErr)
	}
}

func TestFormsRefuseAPerTableRoleThatManagesAnotherTablesViews(t *testing.T) {
	a, _ := formApp(perTableViewManager(tbl2), true)
	a.WorkspaceRoles = tableGrantRoles{}

	_, appErr := a.GetForm(context.Background(), wsID, tbl1, "form-1", accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
}

func TestFormsNeedALicence(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), false)
	ctx := context.Background()

	_, createErr := a.CreateForm(ctx, wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, nil, workspaceAdmin)
	_, getErr := a.GetForm(ctx, wsID, tbl1, "form-1", workspaceAdmin)
	_, updateErr := a.UpdateForm(ctx, wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, workspaceAdmin)
	deleteErr := a.DeleteForm(ctx, wsID, tbl1, "form-1", workspaceAdmin)

	for name, appErr := range map[string]*model.AppError{"create": createErr, "get": getErr, "update": updateErr, "delete": deleteErr} {
		if appErr == nil || appErr.Status != http.StatusPaymentRequired {
			t.Errorf("%s: want 402, got %v", name, appErr)
		}
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestFormsRefuseAManagerFromWhomTheTableIsHidden(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)
	a.WorkspaceRoles = formRoles{hiddenTable: tbl1}
	ctx := context.Background()

	_, createErr := a.CreateForm(ctx, wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, nil, workspaceAdmin)
	_, getErr := a.GetForm(ctx, wsID, tbl1, "form-1", workspaceAdmin)
	_, updateErr := a.UpdateForm(ctx, wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, workspaceAdmin)
	deleteErr := a.DeleteForm(ctx, wsID, tbl1, "form-1", workspaceAdmin)

	for name, appErr := range map[string]*model.AppError{"create": createErr, "get": getErr, "update": updateErr, "delete": deleteErr} {
		if appErr == nil || appErr.Status != http.StatusForbidden {
			t.Errorf("%s: want 403, got %v", name, appErr)
		}
	}
	if ws.writes != 0 || ws.forms["form-1"].Name != "Intake" {
		t.Errorf("the form was changed: %d writes, %+v", ws.writes, ws.forms["form-1"])
	}
}

func TestFormOutsideTheURLWorkspaceOrTableIsRefused(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(a *App, workspaceID, tableID string) *model.AppError{
		"get": func(a *App, workspaceID, tableID string) *model.AppError {
			_, appErr := a.GetForm(ctx, workspaceID, tableID, "form-1", workspaceAdmin)
			return appErr
		},
		"update": func(a *App, workspaceID, tableID string) *model.AppError {
			_, appErr := a.UpdateForm(ctx, workspaceID, tableID, "form-1", model.WorkspaceForm{Name: "Taken over"}, nil, workspaceAdmin)
			return appErr
		},
		"delete": func(a *App, workspaceID, tableID string) *model.AppError {
			return a.DeleteForm(ctx, workspaceID, tableID, "form-1", workspaceAdmin)
		},
	}
	places := map[string]struct {
		workspaceID, tableID string
		want                 int
	}{
		"another workspace": {"workspace-2", tbl1, http.StatusForbidden},
		"another table":     {wsID, tbl2, http.StatusNotFound},
	}

	for callName, call := range calls {
		for placeName, place := range places {
			t.Run(callName+" from "+placeName, func(t *testing.T) {
				a, ws := formApp(workspaceAdminAccess(), true)
				ws.onlyMember = ""

				appErr := call(a, place.workspaceID, place.tableID)
				if appErr == nil || appErr.Status != place.want {
					t.Fatalf("want %d, got %v", place.want, appErr)
				}
				if ws.writes != 0 || ws.forms["form-1"].Name != "Intake" {
					t.Errorf("the form was changed: %d writes, %+v", ws.writes, ws.forms["form-1"])
				}
			})
		}
	}
}

func TestUpdateFormUnknownIsNotFound(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	_, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "missing", model.WorkspaceForm{Name: "Renamed"}, nil, workspaceAdmin)
	if appErr == nil || appErr.Status != http.StatusNotFound {
		t.Fatalf("want 404, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestUpdateFormAnswersWithTheStoredForm(t *testing.T) {
	a, _ := formApp(workspaceAdminAccess(), true)

	request := model.WorkspaceForm{Name: "Renamed", CreatedBy: "someone", PublicToken: "forged"}
	fields := []model.WorkspaceFormField{{ID: "form-2-field", HeaderName: "name", Label: "Name"}}

	updated, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "form-1", request, fields, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("UpdateForm: %v", appErr)
	}

	if updated.Name != "Renamed" || updated.CreatedBy != "creator" || updated.PublicToken != "token-1" {
		t.Errorf("answered %+v, want the stored form renamed", updated.WorkspaceForm)
	}
	if len(updated.Fields) != 1 || updated.Fields[0].ID == "form-2-field" || updated.Fields[0].FormID != "form-1" {
		t.Errorf("answered fields %+v, want one new field of form-1", updated.Fields)
	}
}

func TestFormsRefuseInvalidInput(t *testing.T) {
	long := strings.Repeat("é", 256)
	valid := func() (model.WorkspaceForm, []model.WorkspaceFormField) {
		return model.WorkspaceForm{Name: "Signup", AccentColor: "#4f46e5", CoverPosition: 50},
			[]model.WorkspaceFormField{{HeaderName: "name", Label: "Name", ColumnPosition: "left"}}
	}

	for _, tc := range []struct {
		name   string
		change func(form *model.WorkspaceForm, fields *[]model.WorkspaceFormField)
	}{
		{"empty name", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.Name = " " }},
		{"long name", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.Name = long }},
		{"long title", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.Title = long }},
		{"accent colour not a hex colour", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.AccentColor = "red" }},
		{"cover position below 0", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.CoverPosition = -1 }},
		{"cover position above 100", func(f *model.WorkspaceForm, _ *[]model.WorkspaceFormField) { f.CoverPosition = 101 }},
		{"unknown column", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].HeaderName = "missing" }},
		{"column spelled differently", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].HeaderName = "Name" }},
		{"server-kept column", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].HeaderName = "created_at" }},
		{"column twice", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { *fs = append(*fs, (*fs)[0]) }},
		{"long label", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].Label = long }},
		{"long placeholder", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].Placeholder = long }},
		{"unknown column position", func(_ *model.WorkspaceForm, fs *[]model.WorkspaceFormField) { (*fs)[0].ColumnPosition = "middle" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, ws := formApp(workspaceAdminAccess(), true)
			ctx := context.Background()

			form, fields := valid()
			tc.change(&form, &fields)
			_, createErr := a.CreateForm(ctx, wsID, tbl1, form, fields, workspaceAdmin)

			form, fields = valid()
			tc.change(&form, &fields)
			_, updateErr := a.UpdateForm(ctx, wsID, tbl1, "form-1", form, fields, workspaceAdmin)

			for name, appErr := range map[string]*model.AppError{"create": createErr, "update": updateErr} {
				if appErr == nil || appErr.Status != http.StatusBadRequest {
					t.Errorf("%s: want 400, got %v", name, appErr)
				}
			}
			if ws.writes != 0 {
				t.Errorf("wrote %d times, want none", ws.writes)
			}
		})
	}
}

func TestCreateFormStoresAFieldWithoutPositionAsFullWidth(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	fields := []model.WorkspaceFormField{{HeaderName: "budget", Label: "Budget"}}
	created, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, fields, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("CreateForm: %v", appErr)
	}

	if got := ws.forms[created.ID].Fields[0].ColumnPosition; got != "full" {
		t.Errorf("column position %q, want full", got)
	}
}

type fakeFormChangeStore struct {
	perUserRoleStore
	deleted []string
}

func (f *fakeFormChangeStore) GetTableWorkspaceID(_ context.Context, _ string) (string, error) {
	return wsID, nil
}

func (f *fakeFormChangeStore) GetViewVisibility(_ context.Context, _ string) (bool, string, error) {
	return true, "admin", nil
}

func (f *fakeFormChangeStore) GetForm(_ context.Context, formID string) (*model.WorkspaceFormWithFields, error) {
	return &model.WorkspaceFormWithFields{WorkspaceForm: model.WorkspaceForm{ID: formID, WorkspaceID: wsID, TableID: tbl1}}, nil
}

func (f *fakeFormChangeStore) DeleteForm(_ context.Context, formID string) error {
	f.deleted = append(f.deleted, formID)
	return nil
}

func TestDeleteFormTellsThoseWhoSeeTheTable(t *testing.T) {
	everyone := []string{"admin", "member", "hidden"}
	a, clients := changeApp(everyone, changeRoles{hidden: map[string]bool{"hidden": true}})
	ws := &fakeFormChangeStore{perUserRoleStore: a.Store.Workspace.(perUserRoleStore)}
	a.Store.Workspace = ws
	a.Server.License = &model.License{
		StartsAt:  time.Now().Add(-time.Hour).Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Features:  &model.Features{Forms: boolPtr(true)},
	}

	if appErr := a.DeleteForm(context.Background(), wsID, tbl1, "form-1", workspaceAdmin); appErr != nil {
		t.Fatalf("DeleteForm: %v", appErr)
	}

	if len(ws.deleted) != 1 {
		t.Fatalf("deleted %v, want form-1", ws.deleted)
	}
	for _, id := range []string{"admin", "member"} {
		if data := received(clients[id]); data["type"] != "DELETE_VIEW" || data["delete_id"] != "form-1" {
			t.Errorf("%s received %v, want DELETE_VIEW of form-1", id, data)
		}
	}
	if leaked := received(clients["hidden"]); leaked != nil {
		t.Errorf("someone who cannot see the table was sent %v", leaked)
	}
}

func TestCreateFormRefusesANameAnotherViewHas(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	_, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Intake"}, nil, workspaceAdmin)
	if appErr == nil || appErr.Status != http.StatusConflict || appErr.ID != "workspace.view_already_exists" {
		t.Fatalf("want 409 workspace.view_already_exists, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

type formChangeStore struct {
	perUserRoleStore
	form *model.WorkspaceFormWithFields
}

func (f formChangeStore) GetForm(_ context.Context, formID string) (*model.WorkspaceFormWithFields, error) {
	if formID != f.form.ID {
		return nil, nil
	}

	stored := *f.form
	return &stored, nil
}

func (f formChangeStore) GetTableWorkspaceID(context.Context, string) (string, error) {
	return f.form.WorkspaceID, nil
}

func (f formChangeStore) GetTableName(string) (string, error) {
	return "table_a", nil
}

func (f formChangeStore) GetColumnNames(context.Context, string) ([]string, error) {
	return nil, nil
}

func (f formChangeStore) UpdateForm(context.Context, model.WorkspaceForm, []model.WorkspaceFormField) error {
	return nil
}

func TestUpdateFormTellsItsPagesWhichFormChanged(t *testing.T) {
	a, clients := changeApp([]string{"member"}, changeRoles{})
	a.Store.Workspace = formChangeStore{
		perUserRoleStore: a.Store.Workspace.(perUserRoleStore),
		form:             &model.WorkspaceFormWithFields{WorkspaceForm: model.WorkspaceForm{ID: "form-1", WorkspaceID: "ws-a", TableID: "t-a"}},
	}
	a.Server.License = &model.License{
		StartsAt:  time.Now().Add(-time.Hour).Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Features:  &model.Features{Forms: boolPtr(true)},
	}

	if _, appErr := a.UpdateForm(context.Background(), "ws-a", "t-a", "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, workspaceAdmin); appErr != nil {
		t.Fatalf("UpdateForm: %v", appErr)
	}

	data := received(clients["member"])
	if data["type"] != "FORM_CONFIG_CHANGED" || data["form_id"] != "form-1" || data["workspace_id"] != "ws-a" || data["table_id"] != "t-a" {
		t.Errorf("published %v, want FORM_CONFIG_CHANGED for form-1", data)
	}
}
