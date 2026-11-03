// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

const maxFormImageSize = 5 << 20

var formImageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// SetFormImage stores a form's cover or logo as a workspace attachment of
// the form, served like a row's files, and removes the one it replaces.
func (a *App) SetFormImage(ctx context.Context, workspaceID, tableID, formID, kind string, image *multipart.FileHeader, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if appErr := a.checkFormImageChange(workspaceID, kind, user); appErr != nil {
		return nil, appErr
	}

	if image.Size > maxFormImageSize {
		return nil, model.NewAppError("workspace.form_file_too_large", http.StatusRequestEntityTooLarge)
	}

	ext := strings.ToLower(filepath.Ext(image.Filename))
	if formImageTypes[ext] == "" {
		return nil, model.NewAppError("workspace.form_image_type_not_allowed", http.StatusBadRequest)
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user)
	if appErr != nil {
		return nil, appErr
	}

	f, err := image.Open()
	if err != nil {
		tlog.Errorw("Failed to open an uploaded form image", "form_id", formID, "error", err)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	defer f.Close()

	mimeType, err := sniffContentType(f)
	if err != nil {
		tlog.Errorw("Failed to read an uploaded form image", "form_id", formID, "error", err)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	if mimeType != formImageTypes[ext] {
		return nil, model.NewAppError("workspace.form_image_type_not_allowed", http.StatusBadRequest)
	}

	replaced, appErr := a.formImageAttachment(form, kind)
	if appErr != nil {
		return nil, appErr
	}

	storageID, err := a.primaryStorageID()
	if err != nil {
		tlog.Errorw("Failed to get primary storage", "form_id", formID, "error", err)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	now := time.Now().Unix()
	attachment := model.WorkspaceAttachment{
		ID:          model.NewID(),
		WorkspaceID: workspaceID,
		TableID:     tableID,
		FieldID:     formImageFieldID(kind),
		TaskID:      formID,
		UserID:      user.ID,
		Name:        kind + ext,
		Size:        image.Size,
		MimeType:    mimeType,
		StorageID:   storageID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := a.writeProjectFile(ctx, storageID, attachmentPath(attachment), f, image.Size); err != nil {
		tlog.Errorw("Failed to write a form image",
			"form_id", formID,
			"storage_id", storageID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	if err := a.Store.Workspace.SetFormImage(ctx, formID, kind, &attachment, replaced.ID); err != nil {
		a.removeFormImageFile(ctx, attachment)
		tlog.Errorw("Failed to set a form image",
			"form_id", formID,
			"kind", kind,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_update_failed", http.StatusInternalServerError)
	}

	a.removeFormImageFile(ctx, replaced)

	return a.formImageChanged(ctx, workspaceID, tableID, formID, user)
}

func (a *App) RemoveFormImage(ctx context.Context, workspaceID, tableID, formID, kind string, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	if appErr := a.checkFormImageChange(workspaceID, kind, user); appErr != nil {
		return nil, appErr
	}

	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user)
	if appErr != nil {
		return nil, appErr
	}

	replaced, appErr := a.formImageAttachment(form, kind)
	if appErr != nil {
		return nil, appErr
	}

	if err := a.Store.Workspace.SetFormImage(ctx, formID, kind, nil, replaced.ID); err != nil {
		tlog.Errorw("Failed to remove a form image",
			"form_id", formID,
			"kind", kind,
			"error", err,
		)
		return nil, model.NewAppError("workspace.form_update_failed", http.StatusInternalServerError)
	}

	a.removeFormImageFile(ctx, replaced)

	return a.formImageChanged(ctx, workspaceID, tableID, formID, user)
}

func (a *App) checkFormImageChange(workspaceID, kind string, user model.User) *model.AppError {
	if !a.Server.License.HasForms() {
		return model.NewAppError("license.feature_unavailable", http.StatusPaymentRequired)
	}

	if !a.ProjectWorkspaceHasPermission(user, workspaceID, model.PermissionUpdateWorkspaceView) {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if kind != model.FormImageCover && kind != model.FormImageLogo {
		return model.NewAppError("request.invalid", http.StatusBadRequest)
	}

	return nil
}

// formImageAttachment is the form's current image of the kind, or an empty
// attachment when it has none.
func (a *App) formImageAttachment(form *model.WorkspaceFormWithFields, kind string) (model.WorkspaceAttachment, *model.AppError) {
	id := form.Logo
	if kind == model.FormImageCover {
		id = form.CoverImage
	}

	if id == "" {
		return model.WorkspaceAttachment{}, nil
	}

	attachment, err := a.Store.Workspace.GetAttachment(id, form.WorkspaceID)
	if err != nil {
		tlog.Errorw("Failed to retrieve a form image",
			"form_id", form.ID,
			"file_id", id,
			"error", err,
		)
		return model.WorkspaceAttachment{}, model.NewAppError("workspace.form_get_failed", http.StatusInternalServerError)
	}

	if attachment == nil || attachment.TaskID != form.ID {
		return model.WorkspaceAttachment{ID: id}, nil
	}

	return *attachment, nil
}

// formImages are the form's cover and logo, for removing their files once
// the form is deleted.
func (a *App) formImages(form *model.WorkspaceFormWithFields) ([]model.WorkspaceAttachment, *model.AppError) {
	images := make([]model.WorkspaceAttachment, 0, 2)
	for _, kind := range []string{model.FormImageCover, model.FormImageLogo} {
		image, appErr := a.formImageAttachment(form, kind)
		if appErr != nil {
			return nil, appErr
		}

		images = append(images, image)
	}

	return images, nil
}

// viewFormImages are formImages of the form a view of the table is, or none
// when the view is not a form.
func (a *App) viewFormImages(ctx context.Context, workspaceID, tableID, viewID string) ([]model.WorkspaceAttachment, *model.AppError) {
	form, err := a.Store.Workspace.GetForm(ctx, viewID)
	if err != nil {
		tlog.Errorw("Failed to retrieve the form of a view",
			"view_id", viewID,
			"error", err,
		)
		return nil, model.NewAppError("workspace.table_view_delete_failed", http.StatusInternalServerError)
	}

	if form == nil || form.WorkspaceID != workspaceID || form.TableID != tableID {
		return nil, nil
	}

	return a.formImages(form)
}

func (a *App) removeFormImageFiles(ctx context.Context, images []model.WorkspaceAttachment) {
	for _, image := range images {
		a.removeFormImageFile(ctx, image)
	}
}

func (a *App) removeFormImageFile(ctx context.Context, image model.WorkspaceAttachment) {
	if image.TaskID == "" {
		return
	}

	if err := a.removeProjectFile(ctx, image.StorageID, attachmentPath(image)); err != nil {
		tlog.Warnw("Failed to remove a form image's file",
			"form_id", image.TaskID,
			"file_id", image.ID,
			"error", err,
		)
	}
}

func (a *App) formImageChanged(ctx context.Context, workspaceID, tableID, formID string, user model.User) (*model.WorkspaceFormWithFields, *model.AppError) {
	updated, appErr := a.getTableForm(ctx, workspaceID, tableID, formID, user)
	if appErr != nil {
		return nil, appErr
	}

	a.publishFormChanged(ctx, workspaceID, tableID, formID)

	return updated, nil
}

// formOfImage is the form whose cover or logo the attachment is, or nil.
func (a *App) formOfImage(ctx context.Context, attachment *model.WorkspaceAttachment) *model.WorkspaceFormWithFields {
	if attachment.FieldID != formImageFieldID(model.FormImageCover) && attachment.FieldID != formImageFieldID(model.FormImageLogo) {
		return nil
	}

	form, err := a.Store.Workspace.GetForm(ctx, attachment.TaskID)
	if err != nil {
		tlog.Errorw("Failed to retrieve the form of an image",
			"file_id", attachment.ID,
			"error", err,
		)
		return nil
	}

	if form == nil || form.WorkspaceID != attachment.WorkspaceID || form.TableID != attachment.TableID {
		return nil
	}

	if attachment.ID != form.CoverImage && attachment.ID != form.Logo {
		return nil
	}

	return form
}

// requireFormOpen asks what opening the form by its link asks.
func (a *App) requireFormOpen(ctx context.Context, user model.User, form *model.WorkspaceFormWithFields) *model.AppError {
	if !a.IsWorkspaceMemberOrGroupMember(form.WorkspaceID, user.ID) {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	if appErr := a.requireViewAccess(ctx, user.ID, form.ID); appErr != nil {
		return appErr
	}

	if !a.canReadTable(ctx, user, form.WorkspaceID, form.TableID) {
		return model.NewAppError("workspace.forbidden", http.StatusForbidden)
	}

	return nil
}

func formImageFieldID(kind string) string {
	return "form_" + kind
}
