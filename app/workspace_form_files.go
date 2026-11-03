// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

const (
	maxFormFileSize = 25 << 20
	maxFormFiles    = 10
	maxFileNameLen  = 255
)

var formFileExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".csv": true, ".txt": true, ".rtf": true,
	".zip": true, ".mp4": true, ".mov": true, ".mp3": true, ".wav": true,
}

var errPendingFileMissing = errors.New("pending form file missing")

// pendingFormFile is what an upload through a form's link keeps of a file
// until the form is submitted. It is written next to the file, in the same
// storage, so whichever server takes the submission can read it.
type pendingFormFile struct {
	FormID   string `json:"form_id"`
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// UploadFormFiles keeps files for a form's file fields until the form is
// submitted, which makes them attachments of the new row.
func (a *App) UploadFormFiles(ctx context.Context, token string, files []*multipart.FileHeader, user *model.User, clientIP string) ([]model.ProjectFileMetadata, *model.AppError) {
	ctx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.getFormByToken(ctx, token, user)
	if appErr != nil {
		return nil, appErr
	}

	if !formShows(form, isFileField) {
		return nil, model.NewAppError("request.invalid", http.StatusBadRequest)
	}

	allowed, err := a.formUploadLimiter().Allow(clientIP, token)
	if err != nil {
		tlog.Errorw("Failed to check form upload rate limit", "error", err)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	if !allowed {
		return nil, model.NewAppError("link.rate_limited", http.StatusTooManyRequests)
	}

	if len(files) == 0 || len(files) > maxFormFiles {
		return nil, model.NewAppError("request.invalid", http.StatusBadRequest)
	}

	for _, fh := range files {
		if appErr := checkFormFile(fh); appErr != nil {
			return nil, appErr
		}
	}

	storageID, err := a.primaryStorageID()
	if err != nil {
		tlog.Errorw("Failed to get primary storage", "form_id", form.ID, "error", err)
		return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
	}

	uploaded := make([]model.ProjectFileMetadata, 0, len(files))
	for _, fh := range files {
		file, err := a.savePendingFormFile(ctx, storageID, form, user.ID, fh)
		if err != nil {
			tlog.Errorw("Failed to save a file uploaded to a form",
				"form_id", form.ID,
				"storage_id", storageID,
				"error", err,
			)
			return nil, model.NewAppError("workspace.file_upload_failed", http.StatusInternalServerError)
		}

		uploaded = append(uploaded, file)
	}

	return uploaded, nil
}

func checkFormFile(fh *multipart.FileHeader) *model.AppError {
	if fh.Size > maxFormFileSize {
		return model.NewAppError("workspace.form_file_too_large", http.StatusRequestEntityTooLarge)
	}

	if !formFileExtensions[strings.ToLower(filepath.Ext(fh.Filename))] {
		return model.NewAppError("workspace.form_file_type_not_allowed", http.StatusBadRequest)
	}

	if utf8.RuneCountInString(fh.Filename) > maxFileNameLen {
		return model.NewAppError("request.invalid", http.StatusBadRequest)
	}

	return nil
}

func (a *App) savePendingFormFile(ctx context.Context, storageID string, form *model.WorkspaceFormWithFields, userID string, fh *multipart.FileHeader) (model.ProjectFileMetadata, error) {
	f, err := fh.Open()
	if err != nil {
		return model.ProjectFileMetadata{}, err
	}

	defer f.Close()

	mimeType, err := sniffContentType(f)
	if err != nil {
		return model.ProjectFileMetadata{}, err
	}

	id := model.NewID()
	if err := a.writeProjectFile(ctx, storageID, pendingFormPath(form, id, filepath.Ext(fh.Filename)), f, fh.Size); err != nil {
		return model.ProjectFileMetadata{}, err
	}

	meta, err := json.Marshal(pendingFormFile{FormID: form.ID, UserID: userID, Name: fh.Filename, Size: fh.Size, MimeType: mimeType})
	if err != nil {
		return model.ProjectFileMetadata{}, err
	}

	if err := a.writeProjectFile(ctx, storageID, pendingFormPath(form, id, ".json"), bytes.NewReader(meta), int64(len(meta))); err != nil {
		return model.ProjectFileMetadata{}, err
	}

	return model.ProjectFileMetadata{ID: id, Name: fh.Filename, Size: fh.Size, Type: mimeType}, nil
}

// pendingFormAttachments turns the pending files a submission names for a
// file field into the new row's attachments. A file uploaded by someone
// else or to another form counts as missing.
func (a *App) pendingFormAttachments(ctx context.Context, storageID string, form *model.WorkspaceFormWithFields, task model.FormTask, field model.WorkspaceFormField, ids []string) ([]model.WorkspaceAttachment, error) {
	attachments := make([]model.WorkspaceAttachment, 0, len(ids))
	for _, id := range ids {
		pending, err := a.readPendingFormFile(ctx, storageID, form, id)
		if err != nil {
			return nil, err
		}

		if pending.FormID != form.ID || pending.UserID != task.CreatedBy {
			return nil, errPendingFileMissing
		}

		attachments = append(attachments, model.WorkspaceAttachment{
			ID:          id,
			WorkspaceID: form.WorkspaceID,
			TableID:     form.TableID,
			FieldID:     field.HeaderID,
			TaskID:      task.ID,
			UserID:      task.CreatedBy,
			Name:        pending.Name,
			Size:        pending.Size,
			MimeType:    pending.MimeType,
			StorageID:   storageID,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.CreatedAt,
		})
	}

	return attachments, nil
}

func (a *App) readPendingFormFile(ctx context.Context, storageID string, form *model.WorkspaceFormWithFields, id string) (*pendingFormFile, error) {
	backend, metaPath, err := a.projectFile(storageID, pendingFormPath(form, id, ".json"))
	if err != nil {
		return nil, err
	}

	exists, err := backend.FileExists(ctx, metaPath)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errPendingFileMissing
	}

	r, err := backend.ReadFile(ctx, metaPath)
	if err != nil {
		return nil, err
	}

	defer r.Close()

	var pending pendingFormFile
	if err := json.NewDecoder(r).Decode(&pending); err != nil {
		return nil, err
	}

	return &pending, nil
}

// promoteFormFiles moves a submission's files from the pending area to
// their row. When one cannot be moved, those already moved go back, so the
// submission can be sent again.
func (a *App) promoteFormFiles(ctx context.Context, form *model.WorkspaceFormWithFields, files []model.WorkspaceAttachment) error {
	for i, f := range files {
		err := a.moveProjectFile(ctx, f.StorageID, pendingFormPath(form, f.ID, filepath.Ext(f.Name)), attachmentPath(f))
		if err != nil {
			a.demoteFormFiles(ctx, form, files[:i])
			return err
		}
	}

	return nil
}

func (a *App) demoteFormFiles(ctx context.Context, form *model.WorkspaceFormWithFields, files []model.WorkspaceAttachment) {
	for _, f := range files {
		err := a.moveProjectFile(ctx, f.StorageID, attachmentPath(f), pendingFormPath(form, f.ID, filepath.Ext(f.Name)))
		if err != nil {
			tlog.Warnw("Failed to return a form file to its pending area",
				"form_id", form.ID,
				"file_id", f.ID,
				"error", err,
			)
		}
	}
}

func (a *App) dropPendingFormFiles(ctx context.Context, form *model.WorkspaceFormWithFields, files []model.WorkspaceAttachment) {
	for _, f := range files {
		if err := a.removeProjectFile(ctx, f.StorageID, pendingFormPath(form, f.ID, ".json")); err != nil {
			tlog.Warnw("Failed to remove a submitted form file's pending record",
				"form_id", form.ID,
				"file_id", f.ID,
				"error", err,
			)
		}
	}
}

func pendingFormPath(form *model.WorkspaceFormWithFields, id, ext string) string {
	return form.WorkspaceID + "/" + form.TableID + "/pending/" + id + ext
}

func isFileField(f model.WorkspaceFormField) bool {
	return f.FieldType == "file"
}
