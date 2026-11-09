// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"

	"github.com/twigex/twigex/internal/filestore"
	"github.com/twigex/twigex/model"
)

// Chosen as UploadWorkspaceFiles chooses, so a form's files go where the
// project's other files do.
func (a *App) primaryStorageID() (string, error) {
	primary, err := a.Store.Storage.GetPrimary()
	if err != nil || primary == nil {
		return "", err
	}

	return primary.ID, nil
}

func (a *App) projectFile(storageID, subPath string) (filestore.FileBackend, string, error) {
	if storageID == "" {
		return &filestore.LocalFileBackend{}, path.Join("data", "projects", subPath), nil
	}

	backend, ok := a.FileStorageObjects[storageID]
	if !ok {
		return nil, "", fmt.Errorf("storage %s is not available", storageID)
	}

	filePath, err := a.BuildFilePath(storageID, subPath, model.AppProjects)
	if err != nil {
		return nil, "", err
	}

	return backend, filePath, nil
}

func (a *App) writeProjectFile(ctx context.Context, storageID, subPath string, r io.Reader, size int64) error {
	backend, filePath, err := a.projectFile(storageID, subPath)
	if err != nil {
		return err
	}

	return backend.WriteFile(ctx, filePath, r, size)
}

func (a *App) moveProjectFile(ctx context.Context, storageID, src, dst string) error {
	backend, srcPath, err := a.projectFile(storageID, src)
	if err != nil {
		return err
	}

	_, dstPath, err := a.projectFile(storageID, dst)
	if err != nil {
		return err
	}

	return backend.MoveFile(ctx, srcPath, dstPath)
}

func (a *App) removeProjectFile(ctx context.Context, storageID, subPath string) error {
	backend, filePath, err := a.projectFile(storageID, subPath)
	if err != nil {
		return err
	}

	return backend.RemoveFile(ctx, filePath)
}

// attachmentPath is where getWorkspaceFile looks for an attachment.
func attachmentPath(a model.WorkspaceAttachment) string {
	return a.WorkspaceID + "/" + a.TableID + "/" + a.TaskID + "/" + a.ID + filepath.Ext(a.Name)
}

// The type is read from the file's first bytes; the one the client sent is not
// trusted.
func sniffContentType(f multipart.File) (string, error) {
	buf := make([]byte, 512)

	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return http.DetectContentType(buf[:n]), nil
}
