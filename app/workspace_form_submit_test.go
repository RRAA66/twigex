// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/twigex/twigex/interfaces"
	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

type fakeFormSubmitStore struct {
	*fakeFormStore
	tableWorkspace map[string]string
	options        []string
	members        []string
	createErr      error
	tableGone      bool
	created        []model.FormTask
	recorded       []model.WorkspaceFormSubmission
	submissions    []model.WorkspaceFormSubmission
	linkedRows     int
	linkedLimit    int
	linkedOffset   int
	linkedAccess   model.SQLFilter
	privateView    bool
	viewShares     []string
}

func (f *fakeFormSubmitStore) GetViewVisibility(context.Context, string) (bool, string, error) {
	return !f.privateView, "creator", nil
}

func (f *fakeFormSubmitStore) GetViewSharedUserIDs(string) ([]string, error) {
	return f.viewShares, nil
}

func (f *fakeFormSubmitStore) IsMember(_, userID string) (bool, error) {
	return userID != outsider.ID, nil
}

func (f *fakeFormSubmitStore) GetFormByToken(_ context.Context, token string) (*model.WorkspaceFormWithFields, error) {
	for _, form := range f.forms {
		if form.PublicToken == token && form.Active {
			stored := *form
			return &stored, nil
		}
	}

	return nil, nil
}

func (f *fakeFormSubmitStore) GetSingleSelectOptions(_ context.Context, _ string, ids []string) (map[string]*model.TaskOrderField, error) {
	options := map[string]*model.TaskOrderField{}
	for _, id := range ids {
		if slices.Contains(f.options, id) {
			options[id] = &model.TaskOrderField{ID: id}
		}
	}

	return options, nil
}

func (f *fakeFormSubmitStore) GetMemberUserIDs(_ context.Context, _ string, userIDs []string) ([]string, error) {
	var found []string
	for _, id := range userIDs {
		if slices.Contains(f.members, id) {
			found = append(found, id)
		}
	}

	return found, nil
}

func (f *fakeFormSubmitStore) GetMembers(string) ([]model.WorkspaceMember, error) {
	members := make([]model.WorkspaceMember, 0, len(f.members))
	for _, id := range f.members {
		members = append(members, model.WorkspaceMember{UserID: id})
	}

	return members, nil
}

func (f *fakeFormSubmitStore) GetLinkableTables(context.Context, string, string) ([]model.FormLinkTable, error) {
	return []model.FormLinkTable{{ID: tbl2, Name: "t2"}}, nil
}

func (f *fakeFormSubmitStore) GetLinkedRecordsLite(_ context.Context, _, _ string, _ []string, limit, offset int, access model.SQLFilter) ([]map[string]any, error) {
	f.linkedLimit = limit
	f.linkedOffset = offset
	f.linkedAccess = access

	rows := []map[string]any{}
	for i := range min(f.linkedRows, limit) {
		rows = append(rows, map[string]any{"id": fmt.Sprintf("row-%d", i+1), "name": "Acme"})
	}

	return rows, nil
}

func (f *fakeFormSubmitStore) IsTableSingleSelect(string) (bool, error) {
	return false, nil
}

func (f *fakeFormSubmitStore) CreateFormTask(_ context.Context, _, _ string, task model.FormTask, submission model.WorkspaceFormSubmission) (map[string]any, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	if f.tableGone {
		return nil, nil
	}

	f.created = append(f.created, task)
	f.recorded = append(f.recorded, submission)
	return map[string]any{"id": task.ID}, nil
}

func (f *fakeFormSubmitStore) GetFormSubmissions(context.Context, string) ([]model.WorkspaceFormSubmission, error) {
	return f.submissions, nil
}

func (f *fakeFormSubmitStore) GetTableWorkspaceID(_ context.Context, tableID string) (string, error) {
	return f.tableWorkspace[tableID], nil
}

// The live update after a submission reads the new row back; the tests stop
// it there.
func (f *fakeFormSubmitStore) GetTableMeta(string) (string, bool, error) {
	return "", false, errors.New("not faked")
}

type fakeFormUserStore struct {
	store.UserStore
	users []model.User
	reads int
}

func (f *fakeFormUserStore) GetByIDs(ids []string) ([]model.User, error) {
	f.reads++

	var found []model.User
	for _, u := range f.users {
		if slices.Contains(ids, u.ID) {
			found = append(found, u)
		}
	}

	return found, nil
}

type fakeFormAuthCounter struct {
	store.AuthStore
	counts map[string]int64
}

func (f *fakeFormAuthCounter) Incr(key string, _ time.Duration) (int64, error) {
	f.counts[key]++
	return f.counts[key], nil
}

// noCreateRoles stand in for enterprise roles that show every table but let
// no one create rows.
type noCreateRoles struct{}

func (noCreateRoles) Resolve(interfaces.AccessInput) interfaces.WorkspaceAccess {
	return noCreateAccess{}
}

type noCreateAccess struct{ openAccess }

func (noCreateAccess) CanPerformRowAction(_, _, _ string) bool { return false }

var submitter = &model.User{ID: "submitter", Role: model.SystemUserRoleId}

var outsider = &model.User{ID: "outsider", Role: model.SystemUserRoleId}

func submitApp(fields ...model.WorkspaceFormField) (*App, *fakeFormSubmitStore, *fakeFormUserStore) {
	a, forms := formApp(workspaceAdminAccess(), true)
	forms.forms["form-1"].Fields = fields

	ws := &fakeFormSubmitStore{
		fakeFormStore:  forms,
		tableWorkspace: map[string]string{tbl1: wsID, tbl2: wsID, "table-3": "workspace-2"},
		members:        []string{"member-1", "member-2"},
		linkedRows:     1,
	}
	users := &fakeFormUserStore{users: []model.User{
		{ID: "member-1", Name: "Anna", LastName: "Ozola", Email: "anna@example.com", Photo: "p1"},
		{ID: "member-2", Name: "Jānis", LastName: "Bērziņš", Email: "janis@example.com"},
		{ID: "submitter", Name: "Sam", LastName: "Smith", Email: "sam@example.com"},
	}}

	a.Store.Workspace = ws
	a.Store.User = users
	a.Store.Auth = &fakeFormAuthCounter{counts: map[string]int64{}}
	return a, ws, users
}

func submit(a *App, user *model.User, values ...model.FormValue) (map[string]any, *model.AppError) {
	return a.SubmitForm(context.Background(), "token-1", model.FormSubmitRequest{Fields: values}, user, "10.0.0.1")
}

func textField(name string, required bool) model.WorkspaceFormField {
	return model.WorkspaceFormField{HeaderName: name, FieldType: "text", Visible: true, Required: required}
}

func TestSubmitFormNeedsALoggedInUser(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))

	_, appErr := submit(a, nil, model.FormValue{Name: "name", Value: "Order"})
	if appErr == nil || appErr.Status != http.StatusUnauthorized {
		t.Fatalf("want 401, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestSubmitFormRequiredFieldMissing(t *testing.T) {
	a, ws, _ := submitApp(textField("amount", true))

	_, appErr := submit(a, submitter, model.FormValue{Name: "amount", Value: ""})
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestSubmitFormStoresTheRowWithItsSubmitter(t *testing.T) {
	a, ws, _ := submitApp(textField("amount", true))

	row, appErr := submit(a, submitter, model.FormValue{Name: "amount", Value: "42"})
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	if len(ws.created) != 1 {
		t.Fatalf("created %d rows, want 1", len(ws.created))
	}

	task := ws.created[0]
	if task.Values["amount"] != "42" || task.CreatedBy != submitter.ID || task.CreatedAt == 0 || row["id"] != task.ID {
		t.Errorf("created %+v answering %v, want amount 42 created now by %s", task, row, submitter.ID)
	}

	recorded := ws.recorded[0]
	if recorded.FormID != "form-1" || recorded.TaskID != task.ID || recorded.SubmittedBy != submitter.ID {
		t.Errorf("recorded %+v, want form-1's submission of %s by %s", recorded, task.ID, submitter.ID)
	}
}

func TestSubmitFormHiddenRequiredFieldNotEnforced(t *testing.T) {
	hidden := textField("internal_notes", true)
	hidden.Visible = false
	a, ws, _ := submitApp(textField("name", false), hidden)

	if _, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}); appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}
	if len(ws.created) != 1 {
		t.Errorf("created %d rows, want 1", len(ws.created))
	}
}

func TestSubmitFormKeepsOnlyTheFormsVisibleFields(t *testing.T) {
	hidden := textField("internal_notes", false)
	hidden.Visible = false
	a, ws, _ := submitApp(textField("name", false), hidden, textField("created_by", false))

	_, appErr := submit(a, submitter,
		model.FormValue{Name: "name", Value: "Order"},
		model.FormValue{Name: "internal_notes", Value: "set by the client"},
		model.FormValue{Name: "created_by", Value: "someone-else"},
		model.FormValue{Name: "secret", Value: "not on the form"},
	)
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	values := ws.created[0].Values
	if len(values) != 1 || values["name"] != "Order" {
		t.Errorf("stored %v, want only the name", values)
	}
}

func TestSubmitFormHoneypotIsSilentlyDropped(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))

	request := model.FormSubmitRequest{Fields: []model.FormValue{{Name: "name", Value: "Spam"}}, Honeypot: "http://spam.example"}
	row, appErr := a.SubmitForm(context.Background(), "token-1", request, submitter, "10.0.0.1")
	if appErr != nil || row != nil {
		t.Fatalf("want a silent drop, got %v, %v", row, appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestSubmitFormIsRateLimited(t *testing.T) {
	a, _, _ := submitApp(textField("name", false))

	var appErr *model.AppError
	for range 21 {
		_, appErr = submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
	}

	if appErr == nil || appErr.Status != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %v", appErr)
	}
}

func TestSubmitFormDates(t *testing.T) {
	dateField := func(name string) model.WorkspaceFormField {
		return model.WorkspaceFormField{HeaderName: name, FieldType: "default_date", Visible: true}
	}

	cases := []struct {
		name       string
		start, due string
		wantStatus int
		wantStart  any
	}{
		{"a day", "2026-03-04", "", 0, "2026-03-04"},
		{"a moment", "2026-03-04T22:00:00.000Z", "", 0, "2026-03-04"},
		{"empty", "", "", 0, nil},
		{"not a date", "next week", "", http.StatusBadRequest, nil},
		{"start after due", "2026-03-05", "2026-03-04", http.StatusBadRequest, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws, _ := submitApp(dateField("start_date"), dateField("due_date"))

			_, appErr := submit(a, submitter,
				model.FormValue{Name: "start_date", Value: c.start},
				model.FormValue{Name: "due_date", Value: c.due},
			)
			if c.wantStatus != 0 {
				if appErr == nil || appErr.Status != c.wantStatus {
					t.Fatalf("want %d, got %v", c.wantStatus, appErr)
				}
				return
			}

			if appErr != nil {
				t.Fatalf("SubmitForm: %v", appErr)
			}
			if got := ws.created[0].Values["start_date"]; got != c.wantStart {
				t.Errorf("stored start %v, want %v", got, c.wantStart)
			}
		})
	}
}

func TestSubmitFormLinksAndOptions(t *testing.T) {
	a, ws, _ := submitApp(
		model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true, Required: true},
		model.WorkspaceFormField{HeaderName: "status", FieldType: "status", LinkedID: "status-options", SingleSelect: true, Visible: true},
	)
	ws.options = []string{"open"}

	_, appErr := submit(a, submitter,
		model.FormValue{Name: "customers", Value: "row-1, row-2"},
		model.FormValue{Name: "status", Value: "open"},
	)
	if appErr != nil {
		t.Fatalf("SubmitForm: %v", appErr)
	}

	task := ws.created[0]
	if !slices.Equal(task.Links["customers"], []string{"row-1", "row-2"}) || task.Selects["status"] != "open" || task.Values["status"] != "open" {
		t.Errorf("created %+v, want two customers and status open", task)
	}
}

func TestSubmitFormRefusesUnknownChoices(t *testing.T) {
	cases := []struct {
		name  string
		field model.WorkspaceFormField
		value any
	}{
		{"an option not on the field", model.WorkspaceFormField{HeaderName: "status", FieldType: "status", LinkedID: "status-options", SingleSelect: true, Visible: true}, "gone"},
		{"an assignee outside the workspace", model.WorkspaceFormField{HeaderName: "assignee", FieldType: "default_assignee", Visible: true}, "stranger"},
		{"a link id that is not one", model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", Visible: true}, "row-1,' OR 1=1"},
		{"a value that is not plain", textField("name", false), map[string]any{"id": "x"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws, _ := submitApp(c.field)

			_, appErr := submit(a, submitter, model.FormValue{Name: c.field.HeaderName, Value: c.value})
			if appErr == nil || appErr.Status != http.StatusBadRequest {
				t.Fatalf("want 400, got %v", appErr)
			}
			if len(ws.created) != 0 {
				t.Errorf("created %d rows, want none", len(ws.created))
			}
		})
	}
}

func TestSubmitFormStoreOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		createErr  error
		tableGone  bool
		wantStatus int
	}{
		{"table gone", nil, true, http.StatusNotFound},
		{"linked row gone", model.ErrLinkTargetMissing, false, http.StatusBadRequest},
		{"write failed after the insert", errors.New("kanban sync failed"), false, http.StatusInternalServerError},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", false))
			ws.createErr = c.createErr
			ws.tableGone = c.tableGone

			_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
			if appErr == nil || appErr.Status != c.wantStatus {
				t.Fatalf("want %d, got %v", c.wantStatus, appErr)
			}
		})
	}
}

func TestPublicFormNeedsALoggedInUser(t *testing.T) {
	a, _, _ := submitApp(textField("name", false))

	_, appErr := a.GetPublicForm(context.Background(), "token-1", nil)
	if appErr == nil || appErr.Status != http.StatusUnauthorized {
		t.Fatalf("want 401, got %v", appErr)
	}
}

func TestPublicFormUnknownOrInactiveIsNotFound(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))
	ws.forms["form-1"].Active = false

	for _, token := range []string{"token-1", "missing"} {
		_, appErr := a.GetPublicForm(context.Background(), token, submitter)
		if appErr == nil || appErr.Status != http.StatusNotFound {
			t.Errorf("%s: want 404, got %v", token, appErr)
		}
	}
}

func TestPublicFormReadsMembersInOneBatch(t *testing.T) {
	a, _, users := submitApp(model.WorkspaceFormField{HeaderName: "assignee", FieldType: "default_assignee", Visible: true})

	form, appErr := a.GetPublicForm(context.Background(), "token-1", submitter)
	if appErr != nil {
		t.Fatalf("GetPublicForm: %v", appErr)
	}

	if users.reads != 1 {
		t.Errorf("read users %d times, want once", users.reads)
	}

	want := []model.FormMember{
		{ID: "member-1", Name: "Anna", Lastname: "Ozola", Photo: "p1"},
		{ID: "member-2", Name: "Jānis", Lastname: "Bērziņš"},
	}
	if !slices.Equal(form.Members, want) {
		t.Errorf("members %+v, want %+v", form.Members, want)
	}
	if len(form.LinkTables) != 0 {
		t.Errorf("link tables %+v, want none without a master link field", form.LinkTables)
	}
}

func TestPublicFormWithoutAPickerSharesNoMembers(t *testing.T) {
	a, _, users := submitApp(textField("name", false))

	form, appErr := a.GetPublicForm(context.Background(), "token-1", submitter)
	if appErr != nil {
		t.Fatalf("GetPublicForm: %v", appErr)
	}

	if users.reads != 0 || len(form.Members) != 0 {
		t.Errorf("read users %d times for %+v, want no members", users.reads, form.Members)
	}
}

func TestPublicLinkedItemsOnlyForTheFormsFields(t *testing.T) {
	a, ws, _ := submitApp(model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true})

	_, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", "table-3", "", 1, 10, submitter)
	if appErr == nil || appErr.Status != http.StatusNotFound {
		t.Fatalf("another table: want 404, got %v", appErr)
	}

	items, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", tbl2, "", 1, 500, submitter)
	if appErr != nil {
		t.Fatalf("GetPublicLinkedItems: %v", appErr)
	}
	if ws.linkedLimit != 51 || len(items.Items) != 1 || items.HasMore {
		t.Errorf("asked the store for %d, answered %+v, want a page of at most 50 and one more to tell if there are more", ws.linkedLimit, items)
	}
}

func TestPublicLinkedItemsSaysWhetherMoreFollow(t *testing.T) {
	a, ws, _ := submitApp(model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true})
	ws.linkedRows = 25

	items, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", tbl2, "", 2, 10, submitter)
	if appErr != nil {
		t.Fatalf("GetPublicLinkedItems: %v", appErr)
	}
	if ws.linkedOffset != 10 || len(items.Items) != 10 || !items.HasMore || items.Items[0] != (model.FormLinkedItem{ID: "row-1", Name: "Acme"}) {
		t.Errorf("read from %d, answered %+v; want the second page of 10 with more to follow", ws.linkedOffset, items)
	}
}

func TestPublicLinkedItemsOfATableTheUserCannotReadAreNotFound(t *testing.T) {
	a, _, _ := submitApp(model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true})
	a.WorkspaceRoles = formRoles{hiddenTable: tbl2}

	_, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", tbl2, "", 1, 10, submitter)
	if appErr == nil || appErr.Status != http.StatusNotFound {
		t.Fatalf("want 404, got %v", appErr)
	}
}

func TestPublicLinkedItemsListOnlyTheUsersOwnTasksWhereTheySeeNoOthers(t *testing.T) {
	a, ws, _ := submitApp(model.WorkspaceFormField{HeaderName: "customers", FieldType: "link", LinkedID: "link-table", ParentTableID: tbl2, Visible: true})
	a.WorkspaceRoles = assignedOnlyRoles{tables: map[string]bool{tbl2: true}}

	if _, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", tbl2, "", 1, 10, submitter); appErr != nil {
		t.Fatalf("GetPublicLinkedItems: %v", appErr)
	}
	if len(ws.linkedAccess.Args) != 1 || ws.linkedAccess.Args[0] != submitter.ID {
		t.Errorf("filtered by %+v, want the submitter's own tasks", ws.linkedAccess)
	}
}

func TestPublicLinkedItemsOfAStatusAreItsOptionsNotTheFormsTable(t *testing.T) {
	a, _, _ := submitApp(model.WorkspaceFormField{HeaderName: "status", FieldType: "status", LinkedID: "status-options", SingleSelect: true, ParentTableID: tbl1, Visible: true})

	if _, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", "status-options", "", 1, 10, submitter); appErr != nil {
		t.Errorf("the status options: %v", appErr)
	}

	if _, appErr := a.GetPublicLinkedItems(context.Background(), "token-1", tbl1, "", 1, 10, submitter); appErr == nil || appErr.Status != http.StatusNotFound {
		t.Errorf("the form's own table: want 404, got %v", appErr)
	}
}

func TestFormSubmissionsNamesTheSubmitters(t *testing.T) {
	a, ws, users := submitApp()
	ws.submissions = []model.WorkspaceFormSubmission{
		{ID: "s1", FormID: "form-1", TaskID: "task-1", SubmittedBy: "member-1"},
		{ID: "s2", FormID: "form-1", TaskID: "task-2", SubmittedBy: "member-1"},
		{ID: "s3", FormID: "form-1", TaskID: "task-3", SubmittedBy: "submitter"},
	}

	submissions, appErr := a.GetFormSubmissions(context.Background(), wsID, tbl1, "form-1", workspaceAdmin)
	if appErr != nil {
		t.Fatalf("GetFormSubmissions: %v", appErr)
	}

	if users.reads != 1 {
		t.Errorf("read users %d times, want once", users.reads)
	}
	if submissions[0].SubmitterInfo == nil || submissions[0].SubmitterInfo.Name != "Anna" ||
		submissions[2].SubmitterInfo == nil || submissions[2].SubmitterInfo.Name != "Sam" {
		t.Errorf("submissions %+v, want each submitter named", submissions)
	}
}

func TestFormSubmissionsRefusesAMemberWhoCannotChangeViews(t *testing.T) {
	a, ws, _ := submitApp()
	ws.fakeAccessWorkspaceStore = memberWithoutViewPermissions()

	_, appErr := a.GetFormSubmissions(context.Background(), wsID, tbl1, "form-1", accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
}

func TestFormSubmissionsOfAnotherTableIsNotFound(t *testing.T) {
	places := map[string][2]string{
		"another workspace": {"workspace-2", "table-3"},
		"another table":     {wsID, tbl2},
	}

	for name, place := range places {
		t.Run(name, func(t *testing.T) {
			a, ws, _ := submitApp()
			ws.onlyMember = ""

			_, appErr := a.GetFormSubmissions(context.Background(), place[0], place[1], "form-1", workspaceAdmin)
			if appErr == nil || appErr.Status != http.StatusNotFound {
				t.Fatalf("want 404, got %v", appErr)
			}
		})
	}
}

func TestPublicFormIsOnlyForWorkspaceMembers(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))

	if _, appErr := a.GetPublicForm(context.Background(), "token-1", outsider); appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("open: want 403, got %v", appErr)
	}

	if _, appErr := submit(a, outsider, model.FormValue{Name: "name", Value: "Order"}); appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("submit: want 403, got %v", appErr)
	}

	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestPublicFormFollowsTheViewsVisibility(t *testing.T) {
	cases := []struct {
		name       string
		shares     []string
		wantStatus int
	}{
		{"a private view not shared with the user", []string{"member-1"}, http.StatusNotFound},
		{"a private view shared with the user", []string{submitter.ID}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", false))
			ws.privateView = true
			ws.viewShares = c.shares

			_, openErr := a.GetPublicForm(context.Background(), "token-1", submitter)
			_, submitErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})

			for what, appErr := range map[string]*model.AppError{"open": openErr, "submit": submitErr} {
				if c.wantStatus == 0 && appErr != nil {
					t.Errorf("%s: %v", what, appErr)
				}
				if c.wantStatus != 0 && (appErr == nil || appErr.Status != c.wantStatus) {
					t.Errorf("%s: want %d, got %v", what, c.wantStatus, appErr)
				}
			}
		})
	}
}

func TestPublicFormOnATableTheUserCannotReadIsForbidden(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))
	a.WorkspaceRoles = formRoles{hiddenTable: tbl1}

	if _, appErr := a.GetPublicForm(context.Background(), "token-1", submitter); appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("open: want 403, got %v", appErr)
	}

	if _, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}); appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("submit: want 403, got %v", appErr)
	}

	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestPublicFormSaysWhetherTheUserMaySubmit(t *testing.T) {
	a, _, _ := submitApp(textField("name", false))

	form, appErr := a.GetPublicForm(context.Background(), "token-1", submitter)
	if appErr != nil || !form.CanSubmit {
		t.Errorf("with the right to create rows: %+v, %v; want can_submit", form, appErr)
	}

	a.WorkspaceRoles = noCreateRoles{}

	form, appErr = a.GetPublicForm(context.Background(), "token-1", submitter)
	if appErr != nil || form.CanSubmit {
		t.Errorf("without the right to create rows: %+v, %v; want the form, not to submit", form, appErr)
	}
}

func TestSubmitFormNeedsTheRightToCreateRows(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false))
	a.WorkspaceRoles = noCreateRoles{}

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"})
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}
