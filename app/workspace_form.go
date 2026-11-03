// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

const maxFormTextBytes = 65535

var formAccentColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (a *App) CreateForm(ctx context.Context, workspaceID, tableID string, form model.WorkspaceForm, fields []model.WorkspaceFormField, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if appErr := a.requireFormTableAccess(ctx, user, workspaceID, tableID, model.PermissionCreateWorkspaceView); appErr != nil {
		return nil, appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if appErr := a.validateForm(ctx, tableID, form, fields, "workspace.form_create_failed"); appErr != nil {
		return nil, appErr
	}

	exists, err := a.Store.Workspace.ViewExistsByName(tableID, form.Name)
	if err != nil {
		tlog.Errorw("Failed to check view name",
			"workspace_id", workspaceID,
			"table_id", tableID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_create_failed", http.StatusInternalServerError)
	}

	if exists {
		return nil, model.NewAppError("workspace.view_already_exists", http.StatusConflict)
	}

	now := time.Now().Unix()
	form.ID = model.NewID()
	form.WorkspaceID = workspaceID
	form.TableID = tableID
	form.PublicToken = model.NewID()
	form.Active = true
	form.CreatedBy = user.ID
	form.CreatedAt = now
	form.UpdatedAt = now

	for i := range fields {
		fields[i].ID = model.NewID()
		fields[i].FormID = form.ID
	}

	if err := a.Store.Workspace.CreateForm(ctx, form, fields); err != nil {
		tlog.Errorw("Failed to create form",
			"workspace_id", workspaceID,
			"table_id", tableID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_create_failed", http.StatusInternalServerError)
	}

	created, err := a.Store.Workspace.GetForm(ctx, form.ID)
	if err != nil || created == nil {
		tlog.Errorw("Failed to retrieve created form",
			"form_id", form.ID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	view := model.WorkspaceView{
		ID:          form.ID,
		WorkspaceID: workspaceID,
		TableID:     tableID,
		ViewType:    "form",
		Name:        form.Name,
		CreatedBy:   user.ID,
		CreatedAt:   now,
		IsPublic:    true,
	}
	a.publishViewChange(ctx, view.ID, projectChange{kind: "VIEW_CREATED", workspaceID: workspaceID, tableID: tableID,
		data: map[string]any{"data": view}})

	return created, nil
}

// GetForm serves the form builder, so it asks for the permission to change
// views, as reading a view's shares does.
func (a *App) GetForm(ctx context.Context, workspaceID, tableID, formID string, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if appErr := a.requireFormTableAccess(ctx, user, workspaceID, tableID, model.PermissionUpdateWorkspaceView); appErr != nil {
		return nil, appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	return a.getTableForm(ctx, workspaceID, tableID, formID, user)
}

func (a *App) UpdateForm(ctx context.Context, workspaceID, tableID, formID string, form model.WorkspaceForm, fields []model.WorkspaceFormField, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if appErr := a.requireFormTableAccess(ctx, user, workspaceID, tableID, model.PermissionUpdateWorkspaceView); appErr != nil {
		return nil, appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if _, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user); appErr != nil {
		return nil, appErr
	}

	if appErr := a.validateForm(ctx, tableID, form, fields, "workspace.form_update_failed"); appErr != nil {
		return nil, appErr
	}

	form.ID = formID
	form.UpdatedAt = time.Now().Unix()

	// Fields are replaced as a whole, so ids sent back by the client are not
	// reused: one taken from another form would collide with its row.
	for i := range fields {
		fields[i].ID = model.NewID()
		fields[i].FormID = formID
	}

	if err := a.Store.Workspace.UpdateForm(ctx, form, fields); err != nil {
		tlog.Errorw("Failed to update form",
			"form_id", formID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_update_failed", http.StatusInternalServerError)
	}

	return a.getTableForm(ctx, workspaceID, tableID, formID, user)
}

func (a *App) DeleteForm(ctx context.Context, workspaceID, tableID, formID string, user model.User) *model.AppError {
	if !a.Server.License.HasForms() {
		return model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if appErr := a.requireFormTableAccess(ctx, user, workspaceID, tableID, model.PermissionDeleteTableView); appErr != nil {
		return appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if _, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user); appErr != nil {
		return appErr
	}

	// The audience is read from the view row, so before the delete removes it.
	deleted := projectChange{kind: "DELETE_VIEW", workspaceID: workspaceID, tableID: tableID, data: map[string]any{"delete_id": formID}}
	if audience, err := a.viewAudience(ctx, formID); err == nil {
		deleted.onlyUsers = audience
	}

	if err := a.Store.Workspace.DeleteForm(ctx, formID); err != nil {
		tlog.Errorw("Failed to delete form",
			"form_id", formID,
			"error", err,
		)
		return model.NewAppError("workspace.form_delete_failed", http.StatusInternalServerError)
	}

	a.publishProjectChange(ctx, deleted)

	return nil
}

// validateForm refuses what the form's columns would not hold, and fields
// that are not the table's own columns or that the server keeps, as a 400
// rather than a failed write. A field without a position is full width.
func (a *App) validateForm(ctx context.Context, tableID string, form model.WorkspaceForm, fields []model.WorkspaceFormField, failedID string) *model.AppError {
	invalid := model.NewAppError("request.invalid", http.StatusBadRequest)

	if !validFormSettings(form) {
		return invalid
	}

	tableName, err := a.Store.Workspace.GetTableName(tableID)

	var columns []string
	if err == nil {
		columns, err = a.Store.Workspace.GetColumnNames(ctx, tableName)
	}

	if err != nil {
		tlog.Errorw("Failed to load the form's table columns",
			"table_id", tableID,
			"error", err,
		)
		return model.NewAppError(failedID, http.StatusInternalServerError)
	}

	seen := make(map[string]bool, len(fields))
	for i, f := range fields {
		if f.ColumnPosition == "" {
			fields[i].ColumnPosition = "full"
		}

		if !validFormField(fields[i]) || seen[f.HeaderName] {
			return invalid
		}

		if model.IsServerKeptColumn(f.HeaderName) || !slices.Contains(columns, f.HeaderName) {
			return invalid
		}

		seen[f.HeaderName] = true
	}

	return nil
}

func validFormSettings(form model.WorkspaceForm) bool {
	if strings.TrimSpace(form.Name) == "" || !fitsVarchar(form.Name, form.Title, form.CoverImage, form.Logo) {
		return false
	}

	if len(form.Description) > maxFormTextBytes {
		return false
	}

	if form.AccentColor != "" && !formAccentColor.MatchString(form.AccentColor) {
		return false
	}

	return form.CoverPosition >= 0 && form.CoverPosition <= 100
}

func validFormField(f model.WorkspaceFormField) bool {
	switch f.ColumnPosition {
	case "full", "left", "right":
	default:
		return false
	}

	return len(f.HeaderID) <= 36 && fitsVarchar(f.Label, f.Placeholder) && len(f.Description) <= maxFormTextBytes
}

func fitsVarchar(values ...string) bool {
	for _, v := range values {
		if utf8.RuneCountInString(v) > 255 {
			return false
		}
	}

	return true
}

// requireFormTableAccess asks what the view endpoints ask, the workspace
// permission or the table's manage_views, and also that the user may read the
// table: a role can hide one table from a member who manages views elsewhere.
func (a *App) requireFormTableAccess(ctx context.Context, user model.User, workspaceID, tableID, permission string) *model.AppError {
	if !a.ProjectWorkspaceHasPermission(user, workspaceID, permission) &&
		!a.ProjectWorkspaceTableHasPermission(user, workspaceID, tableID, "manage_views") {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if !a.canReadTable(ctx, user, workspaceID, tableID) {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	return nil
}

// getTableForm answers for a form outside the URL's workspace and table, or
// in a private view the user is not shown, as though it did not exist.
func (a *App) getTableForm(ctx context.Context, workspaceID, tableID, formID string, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	form, err := a.Store.Workspace.GetForm(ctx, formID)
	if err != nil {
		tlog.Errorw("Failed to retrieve form",
			"form_id", formID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	if form == nil || form.WorkspaceID != workspaceID || form.TableID != tableID {
		return nil, model.NewAppError("workspace.form_not_found", http.StatusNotFound)
	}

	if appErr := a.requireViewAccess(ctx, user.ID, formID); appErr != nil {
		return nil, appErr
	}

	return form, nil
}
