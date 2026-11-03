// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/twigex/twigex/model"
)

func shownWhen(field model.WorkspaceFormField, trigger, operator, value string) model.WorkspaceFormField {
	field.VisibilityRule = &model.VisibilityRule{TriggerField: trigger, Operator: operator, Value: value}
	return field
}

func TestSaveFormRefusesAnInvalidCondition(t *testing.T) {
	cases := []struct {
		name   string
		fields []model.WorkspaceFormField
	}{
		{"watches itself", []model.WorkspaceFormField{shownWhen(textField("name", false), "name", "is_empty", "")}},
		{"watches a field not on the form", []model.WorkspaceFormField{textField("name", false), shownWhen(textField("notes", false), "budget", "is_empty", "")}},
		{"unknown operator", []model.WorkspaceFormField{textField("name", false), shownWhen(textField("notes", false), "name", "greater_than", "1")}},
		{"no value", []model.WorkspaceFormField{textField("name", false), shownWhen(textField("notes", false), "name", "equals", "")}},
		{"value too long", []model.WorkspaceFormField{textField("name", false), shownWhen(textField("notes", false), "name", "equals", strings.Repeat("a", 256))}},
		{"loop", []model.WorkspaceFormField{
			shownWhen(textField("name", false), "notes", "is_empty", ""),
			shownWhen(textField("notes", false), "budget", "is_empty", ""),
			shownWhen(textField("budget", false), "name", "is_empty", ""),
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws := formApp(workspaceAdminAccess(), true)

			_, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Intake"}, c.fields, workspaceAdmin)
			if appErr == nil || appErr.Status != http.StatusBadRequest {
				t.Fatalf("UpdateForm: want 400, got %v", appErr)
			}

			_, appErr = a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Intake"}, c.fields, workspaceAdmin)
			if appErr == nil || appErr.Status != http.StatusBadRequest {
				t.Fatalf("CreateForm: want 400, got %v", appErr)
			}

			if ws.writes != 0 {
				t.Errorf("wrote %d times, want none", ws.writes)
			}
		})
	}
}

func TestSaveFormKeepsAValidCondition(t *testing.T) {
	a, ws := formApp(workspaceAdminAccess(), true)

	fields := []model.WorkspaceFormField{
		textField("name", false),
		shownWhen(textField("notes", false), "name", "is_not_empty", "left over"),
		shownWhen(textField("budget", false), "notes", "equals", "big"),
	}
	if _, appErr := a.UpdateForm(context.Background(), wsID, tbl1, "form-1", model.WorkspaceForm{Name: "Intake"}, fields, workspaceAdmin); appErr != nil {
		t.Fatalf("UpdateForm: %v", appErr)
	}

	stored := ws.forms["form-1"].Fields
	if rule := stored[1].VisibilityRule; rule == nil || rule.TriggerField != "name" || rule.Value != "" {
		t.Errorf("notes rule %+v, want one on name without a value", rule)
	}
	if rule := stored[2].VisibilityRule; rule == nil || rule.Value != "big" {
		t.Errorf("budget rule %+v, want its value kept", rule)
	}
}

func TestSubmitFormRequiresAConditionalFieldOnlyWhileShown(t *testing.T) {
	kind := textField("kind", false)
	reason := shownWhen(textField("reason", true), "kind", "equals", "refund")

	a, ws, _ := submitApp(textField("name", false), kind, reason)
	if _, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "kind", Value: "question"}); appErr != nil {
		t.Fatalf("SubmitForm with the field hidden: %v", appErr)
	}
	if len(ws.created) != 1 {
		t.Fatalf("created %d rows, want 1", len(ws.created))
	}

	a, ws, _ = submitApp(textField("name", false), kind, reason)
	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "kind", Value: "refund"})
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("SubmitForm with the field shown: want 400, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestSubmitFormDropsTheValueOfAHiddenField(t *testing.T) {
	a, ws, _ := submitApp(
		textField("name", false),
		model.WorkspaceFormField{HeaderName: "urgent", FieldType: "checkbox", Visible: true},
		shownWhen(textField("deadline_note", false), "urgent", "equals", "true"),
		shownWhen(textField("escalate_to", false), "deadline_note", "is_empty", ""),
	)

	_, appErr := submit(a, submitter,
		model.FormValue{Name: "name", Value: "Order"},
		model.FormValue{Name: "deadline_note", Value: "sent anyway"},
		model.FormValue{Name: "escalate_to", Value: "boss"},
	)
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	values := ws.created[0].Values
	if _, ok := values["deadline_note"]; ok {
		t.Errorf("stored %v, want no deadline note while urgent is unchecked", values)
	}
	if values["escalate_to"] != "boss" {
		t.Errorf("stored %v, want escalate_to, as the note it watches is hidden and so empty", values)
	}
}

func TestSubmitFormComparesLinksByID(t *testing.T) {
	customers := model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true}
	status := model.WorkspaceFormField{HeaderName: "status", FieldType: "status", LinkedID: "status-options", SingleSelect: true, Visible: true}

	cases := []struct {
		name      string
		field     model.WorkspaceFormField
		customers string
		status    string
		wantNote  bool
	}{
		{"one of the linked rows", shownWhen(textField("note", false), "customers", "equals", "row-2"), "row-1, row-2", "", true},
		{"none of the linked rows", shownWhen(textField("note", false), "customers", "equals", "row-3"), "row-1,row-2", "", false},
		{"not a linked row", shownWhen(textField("note", false), "customers", "not_equals", "row-3"), "row-1", "", true},
		{"the chosen option", shownWhen(textField("note", false), "status", "equals", "opt-done"), "", "opt-done", true},
		{"another option", shownWhen(textField("note", false), "status", "equals", "opt-done"), "", "opt-open", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", false), customers, status, c.field)
			ws.options = []string{"opt-done", "opt-open"}

			_, appErr := submit(a, submitter,
				model.FormValue{Name: "name", Value: "Order"},
				model.FormValue{Name: "customers", Value: c.customers},
				model.FormValue{Name: "status", Value: c.status},
				model.FormValue{Name: "note", Value: "hello"},
			)
			if appErr != nil {
				t.Fatalf("SubmitForm: %v", appErr)
			}

			if _, got := ws.created[0].Values["note"]; got != c.wantNote {
				t.Errorf("note stored = %v, want %v", got, c.wantNote)
			}
		})
	}
}
