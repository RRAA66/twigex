// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"testing"

	"github.com/twigex/twigex/model"
)

func (f *fakeFormSubmitStore) GetTableHeaderMeta(string) (*model.TableHeaderMeta, error) {
	return f.meta, nil
}

func defaultsMeta(fieldType string) *model.TableHeaderMeta {
	return &model.TableHeaderMeta{
		Fields:       map[string]model.WorkspaceFieldData{"budget": {Name: "budget", FieldType: fieldType}},
		Links:        map[string]string{"status": "status-options"},
		LinkedTables: map[string]model.LinkedTableMeta{"status-options": {SingleSelect: true}},
	}
}

func TestSubmitFormAnswersAHiddenFieldsDefault(t *testing.T) {
	budget := model.WorkspaceFormField{HeaderName: "budget", FieldType: "number", DefaultValue: "42"}
	a, ws, _ := submitApp(textField("name", true), budget)

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "budget", Value: "1"})
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	if got := ws.created[0].Values["budget"]; got != 42.0 {
		t.Errorf("budget %v, want the hidden default 42 whatever was sent", got)
	}
}

func TestSubmitFormAnswersAReadOnlyFieldsDefault(t *testing.T) {
	source := model.WorkspaceFormField{HeaderName: "description", FieldType: "text", Visible: true, ReadOnly: true, DefaultValue: "Website"}
	a, ws, _ := submitApp(textField("name", true), source)

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "description", Value: "Changed"})
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	if got := ws.created[0].Values["description"]; got != "Website" {
		t.Errorf("description %v, want the read-only default", got)
	}
}

func TestSubmitFormAnswersAMasterLinkDefaultOnATableTheSubmitterDoesNotSee(t *testing.T) {
	cases := map[string]model.WorkspaceFormField{
		"hidden":    {HeaderName: "source", FieldType: "master link", DefaultValue: "table-4"},
		"read-only": {HeaderName: "source", FieldType: "master link", Visible: true, ReadOnly: true, DefaultValue: "table-4"},
	}

	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", true), source)
			a.WorkspaceRoles = formRoles{hiddenTable: "table-4"}

			_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
			if appErr != nil {
				t.Fatalf("SubmitForm: %v", appErr)
			}

			if got := ws.created[0].Values["source"]; got != "table-4" {
				t.Errorf("source %v, want the default table-4", got)
			}
		})
	}
}

func TestSubmitFormLeavesOutADefaultThatNoLongerHolds(t *testing.T) {
	cases := map[string]model.WorkspaceFormField{
		"deleted option":     {HeaderName: "status", LinkedID: "status-options", SingleSelect: true, DefaultValue: "opt-gone"},
		"member who left":    {HeaderName: "assignee", FieldType: "assignee", Visible: true, ReadOnly: true, Required: true, DefaultValue: "user-gone"},
		"deleted table":      {HeaderName: "source", FieldType: "master link", DefaultValue: "table-gone"},
		"deleted linked row": {HeaderName: "client", LinkedID: tbl2, DefaultValue: "row-gone"},
	}

	for name, stale := range cases {
		t.Run(name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", true), stale)

			_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
			if appErr != nil {
				t.Fatalf("SubmitForm: %v", appErr)
			}

			task := ws.created[0]
			if _, ok := task.Values[stale.HeaderName]; ok || len(task.Links[stale.HeaderName]) != 0 {
				t.Errorf("row has %v / %v for %s, want it left out", task.Values, task.Links, stale.HeaderName)
			}
		})
	}
}

func TestSubmitFormKeepsADefaultsRowsThatStillExist(t *testing.T) {
	client := model.WorkspaceFormField{HeaderName: "client", LinkedID: tbl2, DefaultValue: "row-1,row-gone"}
	a, ws, _ := submitApp(textField("name", true), client)
	ws.linkedRows = 1

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	if got := ws.created[0].Links["client"]; len(got) != 1 || got[0] != "row-1" {
		t.Errorf("client links %v, want only row-1", got)
	}
}

func TestSubmitFormTakesAnAnswerOverAVisibleDefault(t *testing.T) {
	note := model.WorkspaceFormField{HeaderName: "description", FieldType: "text", Visible: true, DefaultValue: "Website"}
	a, ws, _ := submitApp(textField("name", true), note)

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "description", Value: "Phone"})
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	if got := ws.created[0].Values["description"]; got != "Phone" {
		t.Errorf("description %v, want what was answered", got)
	}
}

func TestCreateFormRefusesADefaultItsFieldWouldRefuse(t *testing.T) {
	cases := []struct {
		name, fieldType, header, value string
	}{
		{"a number that is not one", "number", "budget", "lots"},
		{"an option the field does not have", "number", "status", "missing"},
		{"a file", "file", "budget", "report.pdf"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, ws, _ := submitApp()
			ws.meta = defaultsMeta(tc.fieldType)
			ws.options = []string{"open"}

			fields := []model.WorkspaceFormField{{HeaderName: tc.header, Label: "Field", DefaultValue: tc.value}}
			_, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, fields, workspaceAdmin)
			if appErr == nil || appErr.Status != http.StatusBadRequest {
				t.Fatalf("want 400, got %v", appErr)
			}
			if ws.writes != 0 {
				t.Errorf("wrote %d times, want none", ws.writes)
			}
		})
	}
}

func TestCreateFormRefusesADecimalDefaultForAWholeNumberField(t *testing.T) {
	a, ws, _ := submitApp()
	ws.meta = defaultsMeta("number")

	fields := []model.WorkspaceFormField{{HeaderName: "budget", Label: "Budget", DefaultValue: "12.5"}}
	_, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, fields, workspaceAdmin)
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
}

func TestCreateFormKeepsAValidDefault(t *testing.T) {
	a, ws, _ := submitApp()
	ws.meta = defaultsMeta("number")
	ws.options = []string{"open"}

	fields := []model.WorkspaceFormField{
		{HeaderName: "budget", Label: "Budget", DefaultValue: "12"},
		{HeaderName: "status", Label: "Status", DefaultValue: "open", ReadOnly: true, Visible: true},
	}
	created, appErr := a.CreateForm(context.Background(), wsID, tbl1, model.WorkspaceForm{Name: "Signup"}, fields, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("CreateForm: %v", appErr)
	}

	saved := ws.forms[created.ID].Fields
	if saved[0].DefaultValue != "12" || saved[1].DefaultValue != "open" || !saved[1].ReadOnly {
		t.Errorf("fields %+v, want both defaults kept", saved)
	}
}

func TestPublicFormLeavesOutAHiddenFieldsDefault(t *testing.T) {
	hidden := model.WorkspaceFormField{HeaderName: "budget", FieldType: "number", DefaultValue: "42"}
	shown := model.WorkspaceFormField{HeaderName: "description", FieldType: "text", Visible: true, DefaultValue: "Hello"}
	a, _, _ := submitApp(hidden, shown)

	form, appErr := a.GetPublicForm(context.Background(), "token-1", submitter)
	if appErr != nil {
		t.Fatalf("GetPublicForm: %v", appErr)
	}

	if form.Fields[0].DefaultValue != "" || form.Fields[1].DefaultValue != "Hello" {
		t.Errorf("fields %+v, want only the shown field's default", form.Fields)
	}
}
