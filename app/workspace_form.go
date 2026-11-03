// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"time"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

func (a *App) CreateForm(ctx context.Context, workspaceID, tableID string, form model.WorkspaceForm, fields []model.WorkspaceFormField, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionCreateWorkspaceView) &&
		!a.ProjectWorkspaceTableHasPermission(user, workspaceID, tableID, "manage_views") {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if appErr := validateFormConditions(fields); appErr != nil {
		return nil, appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

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

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionUpdateWorkspaceView) {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	return a.getTableForm(ctx, workspaceID, tableID, formID, user)
}

func (a *App) UpdateForm(ctx context.Context, workspaceID, tableID, formID string, form model.WorkspaceForm, fields []model.WorkspaceFormField, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if !a.Server.License.HasForms() {
		return nil, model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionUpdateWorkspaceView) {
		return nil, model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if _, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user); appErr != nil {
		return nil, appErr
	}

	if appErr := validateFormConditions(fields); appErr != nil {
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

	updated, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user)
	if appErr != nil {
		return nil, appErr
	}

	// Named by form, so only the edited form's open pages reload.
	a.publishProjectChange(ctx, projectChange{
		kind:        "FORM_CONFIG_CHANGED",
		workspaceID: workspaceID,
		tableID:     tableID,
		data:        map[string]any{"form_id": formID},
	})

	return updated, nil
}

func (a *App) DeleteForm(ctx context.Context, workspaceID, tableID, formID string, user model.User) *model.AppError {
	if !a.Server.License.HasForms() {
		return model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionDeleteTableView) &&
		!a.ProjectWorkspaceTableHasPermission(user, workspaceID, tableID, "manage_views") {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	if _, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user); appErr != nil {
		return appErr
	}

	if err := a.Store.Workspace.DeleteForm(ctx, formID); err != nil {
		tlog.Errorw("Failed to delete form",
			"form_id", formID,
			"error", err,
		)
		return model.NewAppError("workspace.form_delete_failed", http.StatusInternalServerError)
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
