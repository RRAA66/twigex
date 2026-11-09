// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/twigex/twigex/model"
)

func setupFormSubmit(t *testing.T) *workspaceRepository {
	t.Helper()
	w := setupTaskLinks(t)
	cleanTables(t, "workspace_form_submissions", "workspace_form_fields", "workspace_forms", "workspace_view", "workspace_attachments")

	mustExec(t, `INSERT INTO workspace_forms (id, workspace_id, table_id, name, public_token, active, created_by, created_at, updated_at)
		VALUES ('form-1', 'ws', 't-a', 'Intake', 'token-1', 1, 'u1', 1, 1),
		       ('form-2', 'ws', 't-a', 'Closed', 'token-2', 0, 'u1', 1, 1)`)
	return w
}

func formTask(links map[string][]string) (model.FormTask, model.WorkspaceFormSubmission) {
	task := model.FormTask{
		ID:        "new-1",
		CreatedBy: "u1",
		CreatedAt: 100,
		Values:    map[string]any{"name": "From the form"},
		Links:     links,
	}

	return task, model.WorkspaceFormSubmission{ID: "sub-1", FormID: "form-1", TaskID: task.ID, SubmittedBy: "u1", SubmittedAt: 100}
}

func TestCreateFormTaskWritesTheRowLinksAndSubmission(t *testing.T) {
	w := setupFormSubmit(t)

	task, submission := formTask(map[string][]string{"related": {"b2", "b1"}})
	row, err := w.CreateFormTask(context.Background(), "ws", "t-a", task, submission)
	if err != nil {
		t.Fatalf("CreateFormTask: %v", err)
	}

	if row["id"] != "new-1" || row["name"] != "From the form" {
		t.Errorf("answered %v, want the new row", row)
	}

	var createdBy string
	var createdAt int64
	if err := testDB.QueryRow("SELECT created_by, created_at FROM zz_links_a WHERE id = 'new-1'").Scan(&createdBy, &createdAt); err != nil {
		t.Fatal(err)
	}
	if createdBy != "u1" || createdAt != 100 {
		t.Errorf("row created by %q at %d, want u1 at 100", createdBy, createdAt)
	}

	if got := linksOf(t, "new-1"); !slices.Equal(got, []string{"b1", "b2"}) {
		t.Errorf("links %v, want b1 and b2", got)
	}

	if n := countRows(t, testDB, `SELECT COUNT(*) FROM workspace_form_submissions WHERE id = 'sub-1' AND task_id = 'new-1' AND submitted_by = 'u1'`); n != 1 {
		t.Errorf("found %d submission records, want 1", n)
	}
}

func TestCreateFormTaskAttachesItsFiles(t *testing.T) {
	w := setupFormSubmit(t)

	task, submission := formTask(nil)
	task.Attachments = []model.WorkspaceAttachment{{
		ID: "file-1", WorkspaceID: "ws", TableID: "t-a", FieldID: "f-docs", TaskID: task.ID, UserID: "u1",
		Name: "report.pdf", Size: 8, MimeType: "application/pdf", CreatedAt: 100, UpdatedAt: 100,
	}}
	if _, err := w.CreateFormTask(context.Background(), "ws", "t-a", task, submission); err != nil {
		t.Fatalf("CreateFormTask: %v", err)
	}

	got, err := w.GetAttachment("file-1", "ws")
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if got == nil || got.TaskID != "new-1" || got.FieldID != "f-docs" || got.UserID != "u1" || got.StorageID != "" {
		t.Errorf("attachment %+v, want new-1's file by u1 on local disk", got)
	}
}

func TestCreateFormTaskWritesNothingWhenALinkFails(t *testing.T) {
	w := setupFormSubmit(t)

	task, submission := formTask(map[string][]string{"related": {"b1", "missing"}})
	task.Attachments = []model.WorkspaceAttachment{{ID: "file-1", WorkspaceID: "ws", TableID: "t-a", TaskID: task.ID, Name: "a.pdf"}}
	_, err := w.CreateFormTask(context.Background(), "ws", "t-a", task, submission)
	if !errors.Is(err, model.ErrLinkTargetMissing) {
		t.Fatalf("CreateFormTask = %v, want ErrLinkTargetMissing", err)
	}

	if n := countRows(t, testDB, "SELECT COUNT(*) FROM zz_links_a WHERE id = 'new-1'"); n != 0 {
		t.Errorf("kept %d rows, want none", n)
	}
	if n := countRows(t, testDB, "SELECT COUNT(*) FROM workspace_form_submissions"); n != 0 {
		t.Errorf("kept %d submission records, want none", n)
	}
	if n := countRows(t, testDB, "SELECT COUNT(*) FROM workspace_attachments"); n != 0 {
		t.Errorf("kept %d attachments, want none", n)
	}
}

func TestCreateFormTaskRefusesAValueTheColumnCannotHold(t *testing.T) {
	w := setupFormSubmit(t)

	task, submission := formTask(nil)
	task.Values["name"] = strings.Repeat("a", 256)
	_, err := w.CreateFormTask(context.Background(), "ws", "t-a", task, submission)
	if !errors.Is(err, model.ErrValueRejected) {
		t.Fatalf("CreateFormTask = %v, want ErrValueRejected", err)
	}

	if n := countRows(t, testDB, "SELECT COUNT(*) FROM zz_links_a WHERE id = 'new-1'"); n != 0 {
		t.Errorf("kept %d rows, want none", n)
	}
}

func TestCreateFormTaskOnAGoneTableIsNil(t *testing.T) {
	w := setupFormSubmit(t)

	task, submission := formTask(nil)
	row, err := w.CreateFormTask(context.Background(), "other-ws", "t-a", task, submission)
	if err != nil || row != nil {
		t.Errorf("CreateFormTask = %v, %v; want nil, nil", row, err)
	}
}

func TestGetFormByTokenOnlyFindsActiveForms(t *testing.T) {
	w := setupFormSubmit(t)
	ctx := context.Background()

	active, err := w.GetFormByToken(ctx, "token-1")
	if err != nil || active == nil || active.ID != "form-1" {
		t.Errorf("token-1 = %+v, %v; want form-1", active, err)
	}

	closed, err := w.GetFormByToken(ctx, "token-2")
	if err != nil || closed != nil {
		t.Errorf("token-2 = %+v, %v; want nil for an inactive form", closed, err)
	}
}

func TestGetFormByTokenOnADeletedTableIsNil(t *testing.T) {
	w := setupFormSubmit(t)
	mustExec(t, "UPDATE workspace_tables SET deleted_at = 5 WHERE id = 't-a'")

	form, err := w.GetFormByToken(context.Background(), "token-1")
	if err != nil || form != nil {
		t.Errorf("token-1 = %+v, %v; want nil for a form on a deleted table", form, err)
	}
}

func TestGetFormSubmissionsNewestFirst(t *testing.T) {
	w := setupFormSubmit(t)

	mustExec(t, `INSERT INTO workspace_form_submissions (id, form_id, task_id, submitted_by, submitted_at)
		VALUES ('s1', 'form-1', 'a1', 'u1', 1), ('s2', 'form-1', 'a2', NULL, 2), ('s3', 'form-2', 'a2', 'u1', 3)`)

	got, err := w.GetFormSubmissions(context.Background(), "form-1", 10, 0)
	if err != nil {
		t.Fatalf("GetFormSubmissions: %v", err)
	}
	if len(got) != 2 || got[0].ID != "s2" || got[0].SubmittedBy != "" || got[1].ID != "s1" {
		t.Errorf("got %+v, want s2 then s1", got)
	}

	next, err := w.GetFormSubmissions(context.Background(), "form-1", 1, 1)
	if err != nil {
		t.Fatalf("GetFormSubmissions from 1: %v", err)
	}
	if len(next) != 1 || next[0].ID != "s1" {
		t.Errorf("page from 1 = %+v, want s1 alone", next)
	}

	if count, err := w.CountFormSubmissions(context.Background(), "form-1"); err != nil || count != 2 {
		t.Errorf("count = %d, %v; want 2", count, err)
	}
}
