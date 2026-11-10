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
