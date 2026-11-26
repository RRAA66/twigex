// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twigex/twigex/model"
)

func exportCSV(t *testing.T, a *App, filter model.FormSubmissionsFilter) [][]string {
	t.Helper()

	export, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", filter, "UTC", "24h", map[string]string{"submitted_by": "Sent by"}, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("PrepareFormSubmissionsExport: %v", appErr)
	}

	var out bytes.Buffer
	if err := export.WriteCSV(context.Background(), &out, nil); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\xEF\xBB\xBF"))).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	return records
}

func TestExportFormSubmissionsWritesWhoSentEachAndWhen(t *testing.T) {
	a, ws, _ := submitApp()
	sent := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC).Unix()
	ws.submissions = []model.WorkspaceFormSubmission{
		{ID: "s1", FormID: "form-1", TaskID: "task-1", SubmittedBy: "member-1", SubmittedAt: sent},
	}

	records := exportCSV(t, a, model.FormSubmissionsFilter{})

	want := [][]string{{"Sent by", "Submitted at"}, {"Anna Ozola", "04.11.2026 09:30"}}
	if len(records) != len(want) || !slices.Equal(records[0], want[0]) || !slices.Equal(records[1], want[1]) {
		t.Errorf("CSV %q, want %q", records, want)
	}
}

func TestExportFormSubmissionsKeepsAFormulaLabelInert(t *testing.T) {
	a, _, _ := submitApp()

	export, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{}, "UTC", "24h", map[string]string{"submitted_by": "=HYPERLINK(\"http://x\")"}, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("PrepareFormSubmissionsExport: %v", appErr)
	}

	var out bytes.Buffer
	if err := export.WriteCSV(context.Background(), &out, nil); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	header, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\xEF\xBB\xBF"))).Read()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	if want := `'=HYPERLINK("http://x")`; header[0] != want {
		t.Errorf("header %q, want %q", header[0], want)
	}
}

func TestExportFormSubmissionsEndsWhenItStarts(t *testing.T) {
	a, ws, _ := submitApp()
	before := time.Now().Unix()

	exportCSV(t, a, model.FormSubmissionsFilter{From: 100})

	if got := ws.submissionsFilter; got.From != 100 || got.To < before {
		t.Errorf("read with %+v, want from 100 until now", got)
	}
}

func TestExportFormSubmissionsLeavesOutRowsAnAssignedOnlyUserDoesNotSee(t *testing.T) {
	a, ws, _ := submitApp()
	a.WorkspaceRoles = assignedOnlyRoles{tables: map[string]bool{tbl1: true}}

	exportCSV(t, a, model.FormSubmissionsFilter{})

	access := ws.submissionsFilter.Access
	if ws.submissionsFilter.TableName != "tbl_"+tbl1 || len(access.Args) != 1 || access.Args[0] != workspaceAdmin.ID {
		t.Errorf("read with %+v, want the user's own tasks", ws.submissionsFilter)
	}
}

func TestExportFormSubmissionsRefusesAMemberWhoCannotChangeViews(t *testing.T) {
	a, ws, _ := submitApp()
	ws.fakeAccessWorkspaceStore = memberWithoutViewPermissions()
	a.WorkspaceRoles = formRoles{}

	_, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{}, "", "", nil, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
}

func TestExportFormSubmissionsIsOpenToAViewerOfFormsAndGrids(t *testing.T) {
	a, ws, _ := submitApp()
	ws.fakeAccessWorkspaceStore = memberWithPermissions(model.PermissionViewForm, model.PermissionViewGrid)
	a.WorkspaceRoles = formRoles{}

	if _, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{}, "UTC", "24h", nil, accessUser()); appErr != nil {
		t.Fatalf("PrepareFormSubmissionsExport: %v", appErr)
	}
}

func TestExportFormSubmissionsRefusesAViewerWithoutTheGrid(t *testing.T) {
	a, ws, _ := submitApp()
	ws.fakeAccessWorkspaceStore = memberWithPermissions(model.PermissionViewForm)
	a.WorkspaceRoles = formRoles{}

	_, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{}, "", "", nil, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
}

func TestExportFormSubmissionsRefusesAFromDayAfterTheToDay(t *testing.T) {
	a, _, _ := submitApp()

	_, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{From: 200, To: 100}, "UTC", "24h", nil, workspaceAdmin)
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
}

func TestExportFieldLabelTakesThePagesNameForAFieldWithoutALabel(t *testing.T) {
	labels := map[string]string{"field:status": "Statuss"}

	if got := exportFieldLabel(labels, model.WorkspaceFormField{HeaderName: "status", DisplayName: "Status"}); got != "Statuss" {
		t.Errorf("label %q, want the page's name", got)
	}
	if got := exportFieldLabel(nil, model.WorkspaceFormField{HeaderName: "status", DisplayName: "Status"}); got != "Status" {
		t.Errorf("label %q, want the field's name", got)
	}
}

func TestExportFormSubmissionsWorksOutACalculationAsThePageDoes(t *testing.T) {
	a, ws, _ := submitApp()
	ws.submissions = []model.WorkspaceFormSubmission{{ID: "s1", FormID: "form-1", TaskID: "task-1", SubmittedBy: "member-1"}}
	ws.rows = []map[string]any{{"id": "task-1", "net": "1234.56", "rate": "1.21", "total": "1234.56"}}

	export, appErr := a.PrepareFormSubmissionsExport(context.Background(), wsID, tbl1, "form-1", model.FormSubmissionsFilter{}, "UTC", "24h", nil, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("PrepareFormSubmissionsExport: %v", appErr)
	}

	export.fields = append(export.fields, model.WorkspaceHeaders{Name: "total", HeaderUsage: "calculations", Formula: &model.FormulaSpec{
		Name: "PRODUCT",
		Args: []model.FormulaArg{{Type: "field", Value: "net"}, {Type: "field", Value: "rate"}},
	}})
	export.labels = append(export.labels, "Total")

	var out bytes.Buffer
	if err := export.WriteCSV(context.Background(), &out, nil); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\xEF\xBB\xBF"))).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	if got := records[1][2]; got != "1493.82" {
		t.Errorf("total %q, want 1493.82 as the submissions page shows it", got)
	}
}
