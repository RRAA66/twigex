// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/twigex/twigex/config"
	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

type fakeFormStore struct {
	fakeAccessWorkspaceStore
	forms  map[string]*model.WorkspaceFormWithFields
	writes int
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

func memberWithoutViewPermissions() fakeAccessWorkspaceStore {
	role := model.ProjectWorkspaceRole{ID: "reader-id", Name: "reader", Permissions: []string{model.PermissionCreateTask}}
	return fakeAccessWorkspaceStore{memberRoleName: role.Name, roles: []model.ProjectWorkspaceRole{role}}
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

	_, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Renamed"}, nil, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
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

func TestFormOutsideTheURLWorkspaceOrTableIsNotFound(t *testing.T) {
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
	places := map[string][2]string{
		"another workspace": {"workspace-2", tbl1},
		"another table":     {wsID, tbl2},
	}

	for callName, call := range calls {
		for placeName, place := range places {
			t.Run(callName+" from "+placeName, func(t *testing.T) {
				a, ws := formApp(workspaceAdminAccess(), true)
				ws.onlyMember = ""

				appErr := call(a, place[0], place[1])
				if appErr == nil || appErr.Status != http.StatusNotFound {
					t.Fatalf("want 404, got %v", appErr)
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
