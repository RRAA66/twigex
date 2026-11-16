// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/twigex/twigex/model"
)

func resetFormTables(t *testing.T) {
	cleanTables(t,
		"workspace_attachments",
		"workspace_form_fields",
		"workspace_forms",
		"workspace_view",
		"workspace_fields",
		"workspace_relationships",
		"workspace_tables",
	)
}

func seedFormTable(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO workspace_tables (id, workspace_id, name, display_name, linked, single_select, parent_table_id, both_direction_link, second_table_id, folder_id, deleted_at)
		VALUES ('t1', 'w1', 'tasks', 'Tasks', false, false, '', false, '', '', 0),
		       ('t2', 'w1', 'clients', 'Clients', false, true, 't1', false, '', '', 0)`)
	mustExec(t, `INSERT INTO workspace_fields (id, workspace_id, table_id, field_name, field_display_name, field_type, created_at, deleted_at)
		VALUES ('f-budget', 'w1', 't1', 'budget', 'Budget', 'number', 1, 0),
		       ('f-client', 'w1', 't1', 'client', 'Client', 'link', 1, 0)`)
	mustExec(t, `INSERT INTO workspace_relationships (id, workspace_id, table_id, linked_table_id, table_name, created_at)
		VALUES ('r1', 'w1', 't1', 't2', 'client', 1)`)
}

func TestCreateFormWritesItsViewAndReadsFieldsFromTheTable(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	fields := []model.WorkspaceFormField{
		{ID: "ff-1", HeaderID: "f-budget", Label: "Budget", SortOrder: 2},
		{ID: "ff-2", HeaderName: "client", Label: "Client", SortOrder: 1},
		{ID: "ff-3", HeaderName: "status", Label: "Status", SortOrder: 3},
	}
	if err := w.CreateForm(ctx, form, fields); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_view WHERE id = 'form-1' AND view_type = 'form' AND name = 'Intake'`); n != 1 {
		t.Errorf("found %d form views, want 1", n)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got == nil || got.WorkspaceID != "w1" || got.TableID != "t1" || got.PublicToken != "token-1" {
		t.Fatalf("GetForm = %+v, want the created form", got)
	}
	if len(got.Fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(got.Fields))
	}

	client, budget, status := got.Fields[0], got.Fields[1], got.Fields[2]
	if client.LinkedID != "t2" || !client.SingleSelect || client.ParentTableID != "t1" {
		t.Errorf("client field %+v, want a link to single-select t2 under t1", client)
	}
	if budget.HeaderName != "budget" || budget.FieldType != "number" || budget.DisplayName != "Budget" {
		t.Errorf("budget field %+v, want its name and type read through its header id", budget)
	}
	if status.FieldType != "status" {
		t.Errorf("status field %+v, want the system field type", status)
	}
}

func TestGetFormLeavesOutAFieldDeletedFromTheTable(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	fields := []model.WorkspaceFormField{
		{ID: "ff-1", HeaderID: "f-budget", Label: "Budget", SortOrder: 1},
		{ID: "ff-2", HeaderName: "client", Label: "Client", SortOrder: 2},
		{ID: "ff-3", HeaderName: "status", Label: "Status", SortOrder: 3},
	}
	if err := w.CreateForm(ctx, form, fields); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	mustExec(t, `UPDATE workspace_fields SET deleted_at = 5 WHERE id IN ('f-budget', 'f-client')`)

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if len(got.Fields) != 1 || got.Fields[0].HeaderName != "status" {
		t.Errorf("fields %+v, want only status", got.Fields)
	}
}

func TestGetFormUnknownIsNil(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	w := &workspaceRepository{Db: testDB}

	got, err := w.GetForm(context.Background(), "missing")
	if err != nil || got != nil {
		t.Errorf("GetForm = %+v, %v; want nil, nil", got, err)
	}
}

func TestFormFieldKeepsItsVisibilityRule(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	rule := &model.VisibilityRule{TriggerField: "client", Operator: "equals", Value: "row-1"}
	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	fields := []model.WorkspaceFormField{
		{ID: "ff-1", HeaderName: "client", Label: "Client", SortOrder: 1},
		{ID: "ff-2", HeaderName: "budget", Label: "Budget", SortOrder: 2, VisibilityRule: rule},
	}
	if err := w.CreateForm(ctx, form, fields); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got.Fields[0].VisibilityRule != nil {
		t.Errorf("client rule %+v, want none", got.Fields[0].VisibilityRule)
	}
	if got.Fields[1].VisibilityRule == nil || *got.Fields[1].VisibilityRule != *rule {
		t.Errorf("budget rule %+v, want %+v", got.Fields[1].VisibilityRule, rule)
	}
}

func TestFormFieldKeepsItsDefaultAndReadOnly(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	fields := []model.WorkspaceFormField{
		{ID: "ff-1", HeaderName: "budget", Label: "Budget", SortOrder: 1, DefaultValue: "42", ReadOnly: true},
		{ID: "ff-2", HeaderName: "client", Label: "Client", SortOrder: 2},
	}
	if err := w.CreateForm(ctx, form, fields); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}

	budget, client := got.Fields[0], got.Fields[1]
	if budget.DefaultValue != "42" || !budget.ReadOnly {
		t.Errorf("budget %+v, want its default and read-only kept", budget)
	}
	if client.DefaultValue != "" || client.ReadOnly {
		t.Errorf("client %+v, want no default and editable", client)
	}
}

func TestUpdateFormReplacesFieldsAndRenamesItsView(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, []model.WorkspaceFormField{{ID: "ff-1", HeaderName: "budget", Label: "Budget"}}); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	form.Name = "Renamed"
	if err := w.UpdateForm(ctx, form, []model.WorkspaceFormField{{ID: "ff-2", HeaderName: "client", Label: "Client"}}); err != nil {
		t.Fatalf("UpdateForm: %v", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got.Name != "Renamed" || got.UpdatedAt <= 100 {
		t.Errorf("form %+v, want it renamed with a later updated_at", got.WorkspaceForm)
	}
	if len(got.Fields) != 1 || got.Fields[0].ID != "ff-2" {
		t.Errorf("fields %+v, want only ff-2", got.Fields)
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_view WHERE id = 'form-1' AND name = 'Renamed'`); n != 1 {
		t.Errorf("found %d renamed views, want 1", n)
	}
}

func TestFormTakesTheNameOfItsRenamedTab(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, nil); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	if err := w.UpdateViewName("form-1", "w1", "t1", "From the tab menu"); err != nil {
		t.Fatalf("UpdateViewName: %v", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got.Name != "From the tab menu" {
		t.Errorf("name %q, want the tab's name", got.Name)
	}
}

func TestUpdateFormLeavesAFormSavedSinceItWasRead(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, []model.WorkspaceFormField{{ID: "ff-1", HeaderName: "budget", Label: "Budget"}}); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	first := form
	first.Name = "First"
	if err := w.UpdateForm(ctx, first, nil); err != nil {
		t.Fatalf("first UpdateForm: %v", err)
	}

	second := form
	second.Name = "Second"
	if err := w.UpdateForm(ctx, second, nil); !errors.Is(err, model.ErrFormChanged) {
		t.Fatalf("second UpdateForm: got %v, want ErrFormChanged", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got.Name != "First" {
		t.Errorf("form named %q, want the first save kept", got.Name)
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_view WHERE id = 'form-1' AND name = 'First'`); n != 1 {
		t.Errorf("found %d views named First, want 1", n)
	}
}

func TestDeleteFormRemovesItsFieldsAndView(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, []model.WorkspaceFormField{{ID: "ff-1", HeaderName: "budget", Label: "Budget"}}); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	if err := w.DeleteForm(ctx, "form-1"); err != nil {
		t.Fatalf("DeleteForm: %v", err)
	}

	for _, table := range []string{"workspace_forms", "workspace_view"} {
		if n := countRows(t, testDB, "SELECT COUNT(*) FROM `"+table+"` WHERE id = 'form-1'"); n != 0 {
			t.Errorf("%s keeps %d rows of the form", table, n)
		}
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_form_fields WHERE form_id = 'form-1'`); n != 0 {
		t.Errorf("workspace_form_fields keeps %d rows of the form", n)
	}
}

func TestDeleteTableViewDeletesTheFormItShows(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, []model.WorkspaceFormField{{ID: "ff-1", HeaderName: "budget", Label: "Budget"}}); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	logo := &model.WorkspaceAttachment{ID: "img-1", WorkspaceID: "w1", TableID: "t1", FieldID: "form_logo", TaskID: "form-1", UserID: "u1", Name: "logo.png", MimeType: "image/png"}
	if err := w.SetFormImage(ctx, "form-1", model.FormImageLogo, logo, ""); err != nil {
		t.Fatalf("SetFormImage: %v", err)
	}

	if _, err := w.DeleteTableView(ctx, "w1", "t1", "form-1"); err != nil {
		t.Fatalf("DeleteTableView: %v", err)
	}

	for _, table := range []string{"workspace_forms", "workspace_view"} {
		if n := countRows(t, testDB, "SELECT COUNT(*) FROM `"+table+"` WHERE id = 'form-1'"); n != 0 {
			t.Errorf("%s keeps %d rows of the form", table, n)
		}
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_form_fields WHERE form_id = 'form-1'`); n != 0 {
		t.Errorf("workspace_form_fields keeps %d rows of the form", n)
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_attachments WHERE task_id = 'form-1'`); n != 0 {
		t.Errorf("workspace_attachments keeps %d images of the form", n)
	}
}

func TestDeleteTableViewKeepsAFormOfAnotherTable(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, nil); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	if _, err := w.DeleteTableView(ctx, "w1", "t2", "form-1"); err != nil {
		t.Fatalf("DeleteTableView: %v", err)
	}

	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_forms WHERE id = 'form-1'`); n != 1 {
		t.Errorf("found %d forms, want the form kept", n)
	}
}

func TestSetFormImageReplacesTheOldAttachment(t *testing.T) {
	requireDB(t)
	resetFormTables(t)
	seedFormTable(t)
	w := &workspaceRepository{Db: testDB}
	ctx := context.Background()

	form := model.WorkspaceForm{ID: "form-1", WorkspaceID: "w1", TableID: "t1", Name: "Intake", PublicToken: "token-1", Active: true, CreatedBy: "u1", CreatedAt: 100, UpdatedAt: 100}
	if err := w.CreateForm(ctx, form, nil); err != nil {
		t.Fatalf("CreateForm: %v", err)
	}

	image := func(id string) *model.WorkspaceAttachment {
		return &model.WorkspaceAttachment{ID: id, WorkspaceID: "w1", TableID: "t1", FieldID: "form_cover", TaskID: "form-1", UserID: "u1", Name: "cover.png", MimeType: "image/png"}
	}

	if err := w.SetFormImage(ctx, "form-1", model.FormImageCover, image("img-1"), ""); err != nil {
		t.Fatalf("SetFormImage: %v", err)
	}
	if err := w.SetFormImage(ctx, "form-1", model.FormImageCover, image("img-2"), "img-1"); err != nil {
		t.Fatalf("second SetFormImage: %v", err)
	}

	got, err := w.GetForm(ctx, "form-1")
	if err != nil {
		t.Fatalf("GetForm: %v", err)
	}
	if got.CoverImage != "img-2" || got.Logo != "" {
		t.Errorf("cover %q and logo %q, want img-2 and none", got.CoverImage, got.Logo)
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_attachments WHERE task_id = 'form-1'`); n != 1 {
		t.Errorf("form keeps %d images, want 1", n)
	}

	form.CoverImage = "row-file"
	if err := w.UpdateForm(ctx, form, nil); err != nil {
		t.Fatalf("UpdateForm: %v", err)
	}
	if got, _ := w.GetForm(ctx, "form-1"); got.CoverImage != "img-2" {
		t.Errorf("UpdateForm set the cover to %q, want it left as img-2", got.CoverImage)
	}

	if err := w.DeleteForm(ctx, "form-1"); err != nil {
		t.Fatalf("DeleteForm: %v", err)
	}
	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_attachments WHERE task_id = 'form-1'`); n != 0 {
		t.Errorf("deleted form keeps %d images", n)
	}
}
