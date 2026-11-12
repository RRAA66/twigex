// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

const maxLinkedItemsPage = 50

func (a *App) GetPublicForm(ctx context.Context, token string, user *model.User) (*model.PublicForm, *model.AppError) {
	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getFormByToken(ctx, token, user)
	if appErr != nil {
		return nil, appErr
	}

	public := model.PublicForm{
		WorkspaceFormWithFields: *form,
		Members:                 []model.FormMember{},
		LinkTables:              []model.FormLinkTable{},
		CanSubmit:               a.canSubmitForm(*user, form),
	}

	if formShows(form, isAssigneeField) {
		members, appErr := a.formMembers(ctx, form.WorkspaceID)
		if appErr != nil {
			return nil, appErr
		}

		public.Members = members
	}

	if formShows(form, isMasterLinkField) {
		tables, appErr := a.formLinkTables(ctx, *user, form)
		if appErr != nil {
			return nil, appErr
		}

		public.LinkTables = tables
	}

	return &public, nil
}

// Only the tables of its workspace the user may see.
func (a *App) formLinkTables(ctx context.Context, user model.User, form *model.WorkspaceFormWithFields) ([]model.FormLinkTable, *model.AppError) {
	tables, err := a.Store.Workspace.GetLinkableTables(ctx, form.WorkspaceID, form.TableID)
	if err != nil {
		tlog.Errorw("Failed to retrieve tables for a form",
			"form_id", form.ID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	candidates := make([]model.WorkspaceTable, 0, len(tables))
	for _, t := range tables {
		candidates = append(candidates, model.WorkspaceTable{ID: t.ID})
	}

	visible := map[string]bool{}
	for _, t := range a.resolveAccess(a.buildAccessInput(user, form.WorkspaceID)).VisibleTables(candidates) {
		visible[t.ID] = true
	}

	return slices.DeleteFunc(tables, func(t model.FormLinkTable) bool { return !visible[t.ID] }), nil
}

// GetPublicLinkedItems pages through the rows a link or single-select field
// of the form can pick, by id and name only.
func (a *App) GetPublicLinkedItems(ctx context.Context, token, tableID, search string, page, limit int, user *model.User) (*model.LinkedItemsResponse, *model.AppError) {
	if limit <= 0 || limit > maxLinkedItemsPage {
		limit = maxLinkedItemsPage
	}

	if page < 1 {
		page = 1
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getFormByToken(ctx, token, user)
	if appErr != nil {
		return nil, appErr
	}

	i := slices.IndexFunc(form.Fields, func(f model.WorkspaceFormField) bool {
		return f.Visible && f.LinkedID != "" && pickedTable(f) == tableID
	})
	if i == -1 {
		return nil, model.NewAppError("workspace.form_linked_not_found", http.StatusNotFound)
	}

	// A single-select's options belong to the form's table, which the user
	// already reads; a link's rows are another table's.
	if !form.Fields[i].SingleSelect && !a.canReadTable(ctx, *user, form.WorkspaceID, tableID) {
		return nil, model.NewAppError("workspace.form_linked_not_found", http.StatusNotFound)
	}

	tableName, err := a.Store.Workspace.GetTableName(tableID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.NewAppError("workspace.form_linked_not_found", http.StatusNotFound)
	}

	if err != nil {
		tlog.Errorw("Failed to retrieve the linked table of a form",
			"form_id", form.ID,
			"table_id", tableID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_linked_failed", http.StatusInternalServerError)
	}

	// One row past the page tells whether another page follows.
	access := a.assignedOnlyFilter(*user, form.WorkspaceID, tableID)
	rows, err := a.Store.Workspace.GetLinkedRecordsLite(ctx, tableName, search, nil, limit+1, (page-1)*limit, access)
	if err != nil {
		tlog.Errorw("Failed to retrieve linked items for a form",
			"form_id", form.ID,
			"table_id", tableID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_linked_failed", http.StatusInternalServerError)
	}

	response := model.LinkedItemsResponse{Items: []model.FormLinkedItem{}, HasMore: len(rows) > limit}
	for _, row := range rows[:min(len(rows), limit)] {
		id, _ := row["id"].(string)
		name, _ := row["name"].(string)
		response.Items = append(response.Items, model.FormLinkedItem{ID: id, Name: name})
	}

	return &response, nil
}

// A filled honeypot is dropped without an error and answered with a made-up id,
// so a bot cannot tell it was caught.
func (a *App) SubmitForm(ctx context.Context, token string, request model.FormSubmitRequest, user *model.User) (string, *model.AppError) {
	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getFormByToken(ctx, token, user)
	if appErr != nil {
		return "", appErr
	}

	if !a.canSubmitForm(*user, form) {
		return "", model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if request.Honeypot != "" {
		return model.NewID(), nil
	}

	allowed, err := a.formSubmitLimiter().Allow(user.ID, token)
	if err != nil {
		tlog.Errorw("Failed to check form submit rate limit", "error", err)
		return "", model.NewAppError("workspace.form_submit_failed", http.StatusInternalServerError)
	}

	if !allowed {
		return "", model.NewAppError("link.rate_limited", http.StatusTooManyRequests)
	}

	fields := make(map[string]model.WorkspaceFormField, len(form.Fields))
	for _, f := range form.Fields {
		if f.Visible && f.FieldType != "file" && !model.IsServerKeptColumn(f.HeaderName) {
			fields[f.HeaderName] = f
		}
	}

	now := time.Now().Unix()
	task := model.FormTask{
		ID:        model.NewID(),
		CreatedBy: user.ID,
		CreatedAt: now,
		Values:    map[string]any{},
		Links:     map[string][]string{},
		Selects:   map[string]string{},
	}

	linkTables := map[string]bool{}
	if formShows(form, isMasterLinkField) {
		tables, appErr := a.formLinkTables(ctx, *user, form)
		if appErr != nil {
			return "", appErr
		}

		for _, t := range tables {
			linkTables[t.ID] = true
		}
	}

	for _, v := range request.Fields {
		field, ok := fields[v.Name]
		if !ok {
			continue
		}

		if appErr := a.setFormValue(ctx, form.WorkspaceID, field, v.Value, linkTables, &task); appErr != nil {
			return "", appErr
		}
	}

	for _, f := range fields {
		if !f.Required {
			continue
		}

		if isCheckboxField(f) {
			if task.Values[f.HeaderName] != 1 {
				return "", model.NewAppError("workspace.form_submit_required_field", http.StatusBadRequest)
			}

			continue
		}

		value, ok := task.Values[f.HeaderName]
		if (!ok || value == nil || value == "") && len(task.Links[f.HeaderName]) == 0 {
			return "", model.NewAppError("workspace.form_submit_required_field", http.StatusBadRequest)
		}
	}

	start, _ := task.Values["start_date"].(string)
	due, _ := task.Values["due_date"].(string)
	if start != "" && due != "" && start > due {
		return "", model.NewAppError("workspace.task_dates_reversed", http.StatusBadRequest)
	}

	submission := model.WorkspaceFormSubmission{
		ID:          model.NewID(),
		FormID:      form.ID,
		TaskID:      task.ID,
		SubmittedBy: user.ID,
		SubmittedAt: now,
	}

	row, err := a.Store.Workspace.CreateFormTask(ctx, form.WorkspaceID, form.TableID, task, submission)
	switch {
	case errors.Is(err, model.ErrLinkTargetMissing), errors.Is(err, model.ErrValueRejected):
		return "", model.NewAppError("request.invalid", http.StatusBadRequest)
	case errors.Is(err, sql.ErrNoRows):
		return "", model.NewAppError("workspace.view_or_field_not_found", http.StatusNotFound)
	case err != nil:
		tlog.Errorw("Failed to create a row from a form",
			"workspace_id", form.WorkspaceID,
			"table_id", form.TableID,
			"form_id", form.ID,
			"error", err,
		)
		return "", model.NewAppError("workspace.form_submit_failed", http.StatusInternalServerError)
	case row == nil:
		return "", model.NewAppError("workspace.form_not_found", http.StatusNotFound)
	}

	a.publishTaskRow(ctx, "task_created", form.WorkspaceID, form.TableID, task.ID)

	return task.ID, nil
}

// GetFormSubmissions lists who submitted a form and which row each made.
// Rows are table data, so it asks for reading the table as well as what
// reading the form asks for.
func (a *App) GetFormSubmissions(ctx context.Context, workspaceID, tableID, formID string, user model.User) ([]model.WorkspaceFormSubmission, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionUpdateWorkspaceView) ||
		!a.canReadTable(ctx, user, workspaceID, tableID) {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if _, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user); appErr != nil {
		return nil, appErr
	}

	submissions, err := a.Store.Workspace.GetFormSubmissions(ctx, formID)
	if err != nil {
		tlog.Errorw("Failed to retrieve form submissions",
			"form_id", formID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_submissions_failed", http.StatusInternalServerError)
	}

	var submitterIDs []string
	for _, s := range submissions {
		if s.SubmittedBy != "" && !slices.Contains(submitterIDs, s.SubmittedBy) {
			submitterIDs = append(submitterIDs, s.SubmittedBy)
		}
	}

	users, err := a.Store.User.GetByIDs(submitterIDs)
	if err != nil {
		tlog.Errorw("Failed to retrieve form submitters",
			"form_id", formID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_submissions_failed", http.StatusInternalServerError)
	}

	submitters := make(map[string]model.FormMember, len(users))
	for _, u := range users {
		submitters[u.ID] = formMember(u)
	}

	for i, s := range submissions {
		if info, ok := submitters[s.SubmittedBy]; ok {
			submissions[i].SubmitterInfo = &info
		}
	}

	return submissions, nil
}

// Open to whoever would see its view in the grid: a member of its workspace who
// is shown the view and may read its table.
func (a *App) getFormByToken(ctx context.Context, token string, user *model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if user == nil {
		return nil, model.NewAppError("workspace.form_login_required", http.StatusUnauthorized)
	}

	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	form, err := a.Store.Workspace.GetFormByToken(ctx, token)
	if err != nil {
		tlog.Errorw("Failed to retrieve form by its link", "error", err)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	if form == nil {
		return nil, model.NewAppError("workspace.form_not_found", http.StatusNotFound)
	}

	if !a.IsWorkspaceMemberOrGroupMember(form.WorkspaceID, user.ID) {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if appErr := a.requireViewAccess(ctx, user.ID, form.ID); appErr != nil {
		return nil, appErr
	}

	if !a.canReadTable(ctx, *user, form.WorkspaceID, form.TableID) {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	return form, nil
}

func (a *App) canSubmitForm(user model.User, form *model.WorkspaceFormWithFields) bool {
	return a.CanPerformRowAction(user, form.WorkspaceID, form.TableID, model.PermissionCreateTask, model.PermissionCreateTask)
}

// An empty value leaves the column to its default.
func (a *App) setFormValue(ctx context.Context, workspaceID string, field model.WorkspaceFormField, value any, linkTables map[string]bool, task *model.FormTask) *model.AppError {
	invalid := model.NewAppError("request.invalid", http.StatusBadRequest)

	switch value.(type) {
	case nil, string, float64, bool:
	default:
		return invalid
	}

	text, isText := value.(string)
	name := field.HeaderName

	switch {
	case field.LinkedID != "" && field.SingleSelect:
		if text == "" || text == "0" {
			return nil
		}

		options, err := a.Store.Workspace.GetSingleSelectOptions(ctx, field.LinkedID, []string{text})
		if err != nil {
			tlog.Errorw("Failed to check a form's single-select option",
				"field", name,
				"error", err,
			)
			return model.NewAppError("workspace.form_submit_failed", http.StatusInternalServerError)
		}

		if options[text] == nil {
			return invalid
		}

		task.Values[name] = text
		task.Selects[name] = text

	case field.LinkedID != "":
		var ids []string
		for _, id := range strings.Split(text, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}

			if !validLinkID(id) {
				return invalid
			}

			ids = append(ids, id)
		}

		if len(ids) > maxLinkChanges {
			return invalid
		}

		if len(ids) > 0 {
			task.Links[name] = ids
		}

	case field.FieldType == "link":
		return nil

	case field.FieldType == "default_date" || field.FieldType == "date":
		if value != nil && !isText {
			return invalid
		}

		day, ok := taskDay(text)
		if !ok {
			return invalid
		}

		if day != "" {
			task.Values[name] = day
		}

	case isAssigneeField(field):
		if text == "" {
			return nil
		}

		members, err := a.Store.Workspace.GetMemberUserIDs(ctx, workspaceID, []string{text})
		if err != nil {
			tlog.Errorw("Failed to check the assignee's membership",
				"workspace_id", workspaceID,
				"user_id", text,
				"error", err,
			)
			return model.NewAppError("workspace.form_submit_failed", http.StatusInternalServerError)
		}

		if len(members) == 0 {
			return invalid
		}

		task.Values[name] = text

	case isCheckboxField(field):
		checked := false
		switch v := value.(type) {
		case string:
			checked = v != "" && v != "false" && v != "0"
		case bool:
			checked = v
		case float64:
			checked = v != 0
		}

		task.Values[name] = 0
		if checked {
			task.Values[name] = 1
		}

	case value == nil || isText && strings.TrimSpace(text) == "":
		return nil

	case field.FieldType == "number" || field.FieldType == "decimal":
		number, ok := formNumber(value)
		if !ok {
			return invalid
		}

		task.Values[name] = number

	case isMasterLinkField(field):
		if !linkTables[text] {
			return invalid
		}

		task.Values[name] = text

	case field.FieldType == "url" && (!isText || !safeLink(text)):
		return invalid

	case isText && !fitsTextColumn(name, text):
		return invalid

	default:
		task.Values[name] = value
	}

	return nil
}

func (a *App) formMembers(ctx context.Context, workspaceID string) ([]model.FormMember, *model.AppError) {
	members, err := a.Store.Workspace.GetMembers(workspaceID)
	if err != nil {
		tlog.Errorw("Failed to retrieve members for a form",
			"workspace_id", workspaceID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}

	users, err := a.Store.User.GetByIDs(ids)
	if err != nil {
		tlog.Errorw("Failed to retrieve members for a form",
			"workspace_id", workspaceID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	result := make([]model.FormMember, 0, len(users))
	for _, u := range users {
		result = append(result, formMember(u))
	}

	return result, nil
}

func formMember(u model.User) model.FormMember {
	return model.FormMember{ID: u.ID, Name: u.Name, Lastname: u.LastName, Photo: u.Photo}
}

func formShows(form *model.WorkspaceFormWithFields, match func(model.WorkspaceFormField) bool) bool {
	return slices.ContainsFunc(form.Fields, func(f model.WorkspaceFormField) bool {
		return f.Visible && match(f)
	})
}

func isAssigneeField(f model.WorkspaceFormField) bool {
	return f.FieldType == "assignee" || f.FieldType == "default_assignee"
}

func isMasterLinkField(f model.WorkspaceFormField) bool {
	return f.FieldType == "master link"
}

func isCheckboxField(f model.WorkspaceFormField) bool {
	return f.FieldType == "bool" || f.FieldType == "checkbox"
}

// formNumber reads a number field's value, sent as a number or as text.
func formNumber(value any) (float64, bool) {
	var number float64
	switch v := value.(type) {
	case float64:
		number = v
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}

		number = parsed
	default:
		return 0, false
	}

	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

// fitsTextColumn reports whether text fits its column: a task's name is a
// VARCHAR(255) and every other text column a TEXT.
func fitsTextColumn(column, text string) bool {
	if column == "name" {
		return fitsVarchar(text)
	}

	return len(text) <= maxTaskDescription
}

// A single-select's options table has the field's own table as its parent, so
// the parent is the target only for a link.
func pickedTable(f model.WorkspaceFormField) string {
	if f.SingleSelect || f.ParentTableID == "" {
		return f.LinkedID
	}

	return f.ParentTableID
}
