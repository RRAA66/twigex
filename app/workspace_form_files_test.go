// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/twigex/twigex/internal/filestore"
	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/store"
)

type fakeFormStorageStore struct {
	store.StorageStore
	primary *model.Storage
}

func (f *fakeFormStorageStore) GetPrimary() (*model.Storage, error) {
	return f.primary, nil
}

func (f *fakeFormStorageStore) GetByID(id string) (*model.Storage, error) {
	if f.primary == nil || f.primary.ID != id {
		return nil, errors.New("no such storage")
	}

	return f.primary, nil
}

// useLocalProjectFiles runs the test where project files fall back to when
// no storage is configured, and answers that directory.
func useLocalProjectFiles(t *testing.T, a *App) string {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)
	a.Store.Storage = &fakeFormStorageStore{}

	return filepath.Join(dir, "data", "projects")
}

// usePrimaryStorage gives the app a configured local storage, as an
// administrator would set one up, and answers where its project files go.
func usePrimaryStorage(t *testing.T, a *App) string {
	t.Helper()

	t.Chdir(t.TempDir())

	primary := &model.Storage{ID: "disk-a", Directory: t.TempDir()}
	a.Store.Storage = &fakeFormStorageStore{primary: primary}
	a.FileStorageObjects = map[string]filestore.FileBackend{primary.ID: &filestore.LocalFileBackend{}}

	return filepath.Join(primary.Directory, "projects")
}

func uploadedFiles(t *testing.T, key string, files map[string]string) []*multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for name, content := range files {
		part, err := w.CreateFormFile(key, name)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}

		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}

	return req.MultipartForm.File[key]
}

func fileField(name string, required bool) model.WorkspaceFormField {
	return model.WorkspaceFormField{HeaderID: "field-" + name, HeaderName: name, FieldType: "file", Visible: true, Required: required}
}

// uploadFiles checks the upload and then stores the files, as the endpoint
// does.
func uploadFiles(a *App, user *model.User, files []*multipart.FileHeader) ([]model.ProjectFileMetadata, *model.AppError) {
	form, appErr := a.CheckFormUpload(context.Background(), "token-1", user, "10.0.0.1")
	if appErr != nil {
		return nil, appErr
	}

	return a.UploadFormFiles(context.Background(), form, files, *user)
}

func uploadToForm(t *testing.T, a *App, user *model.User, name, content string) model.ProjectFileMetadata {
	t.Helper()

	uploaded, appErr := uploadFiles(a, user, uploadedFiles(t, "files", map[string]string{name: content}))
	if appErr != nil {
		t.Fatalf("UploadFormFiles: %v", appErr)
	}

	if len(uploaded) != 1 {
		t.Fatalf("uploaded %d files, want 1", len(uploaded))
	}

	return uploaded[0]
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(content)
}

func TestUploadFormFilesRefusesANonMember(t *testing.T) {
	a, _, _ := submitApp(textField("name", false), fileField("docs", false))
	root := useLocalProjectFiles(t, a)

	files := uploadedFiles(t, "files", map[string]string{"report.pdf": "%PDF-1.4"})

	_, appErr := uploadFiles(a, outsider, files)
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("wrote files for a non-member: %v", err)
	}
}

func TestUploadFormFilesNeedsTheRightToCreateRows(t *testing.T) {
	a, _, _ := submitApp(fileField("docs", false))
	a.WorkspaceRoles = noCreateRoles{}

	_, appErr := a.CheckFormUpload(context.Background(), "token-1", submitter, "10.0.0.1")
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}
}

func TestUploadFormFilesRefusesATypeNotAllowed(t *testing.T) {
	a, _, _ := submitApp(fileField("docs", false))
	useLocalProjectFiles(t, a)

	files := uploadedFiles(t, "files", map[string]string{"run.exe": "MZ"})

	_, appErr := uploadFiles(a, submitter, files)
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
}

func TestUploadFormFilesNeedsAFileFieldOnTheForm(t *testing.T) {
	a, _, _ := submitApp(textField("name", false))
	useLocalProjectFiles(t, a)

	files := uploadedFiles(t, "files", map[string]string{"report.pdf": "%PDF-1.4"})

	_, appErr := uploadFiles(a, submitter, files)
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
}

func TestSubmitFormAttachesUploadedFilesToTheRow(t *testing.T) {
	cases := []struct {
		name    string
		storage func(*testing.T, *App) string
		storeID string
	}{
		{"local data directory", useLocalProjectFiles, ""},
		{"primary storage", usePrimaryStorage, "disk-a"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, ws, _ := submitApp(textField("name", false), fileField("docs", true))
			root := tc.storage(t, a)

			uploaded := uploadToForm(t, a, submitter, "Report.PDF", "%PDF-1.4 report")

			_, appErr := submit(a, submitter,
				model.FormValue{Name: "name", Value: "Order"},
				model.FormValue{Name: "docs", Value: uploaded.ID},
			)
			if appErr != nil {
				t.Fatalf("SubmitForm: %v", appErr)
			}

			task := ws.created[0]
			if len(task.Attachments) != 1 {
				t.Fatalf("attached %d files, want 1", len(task.Attachments))
			}

			got := task.Attachments[0]
			want := model.WorkspaceAttachment{
				ID: uploaded.ID, WorkspaceID: wsID, TableID: tbl1, FieldID: "field-docs", TaskID: task.ID,
				UserID: submitter.ID, Name: "Report.PDF", Size: int64(len("%PDF-1.4 report")), MimeType: "application/pdf",
				StorageID: tc.storeID, CreatedAt: task.CreatedAt, UpdatedAt: task.CreatedAt,
			}
			if got != want {
				t.Errorf("attached %+v, want %+v", got, want)
			}

			if content := readFile(t, filepath.Join(root, wsID, tbl1, task.ID, uploaded.ID+".PDF")); content != "%PDF-1.4 report" {
				t.Errorf("row's file holds %q", content)
			}

			pending, err := os.ReadDir(filepath.Join(root, wsID, tbl1, "pending"))
			if err != nil || len(pending) != 0 {
				t.Errorf("pending area holds %v (%v), want it empty", pending, err)
			}
		})
	}
}

func TestSubmitFormRequiresARequiredFileField(t *testing.T) {
	a, ws, _ := submitApp(textField("name", false), fileField("docs", true))
	useLocalProjectFiles(t, a)

	_, appErr := submit(a, submitter, model.FormValue{Name: "name", Value: "Order"}, model.FormValue{Name: "docs", Value: ""})
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

func TestSubmitFormRefusesAFileSomeoneElseUploaded(t *testing.T) {
	a, ws, _ := submitApp(fileField("docs", false))
	root := useLocalProjectFiles(t, a)

	uploaded := uploadToForm(t, a, &model.User{ID: "member-1"}, "report.pdf", "%PDF-1.4")

	_, appErr := submit(a, submitter, model.FormValue{Name: "docs", Value: uploaded.ID})
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
	if content := readFile(t, filepath.Join(root, wsID, tbl1, "pending", uploaded.ID+".pdf")); content != "%PDF-1.4" {
		t.Errorf("pending file holds %q", content)
	}
}

func TestSubmitFormRefusesAnUnknownFile(t *testing.T) {
	a, ws, _ := submitApp(fileField("docs", false))
	useLocalProjectFiles(t, a)

	for _, value := range []string{"never-uploaded", "../../secret"} {
		_, appErr := submit(a, submitter, model.FormValue{Name: "docs", Value: value})
		if appErr == nil || appErr.Status != http.StatusBadRequest {
			t.Errorf("%q: want 400, got %v", value, appErr)
		}
	}
	if len(ws.created) != 0 {
		t.Errorf("created %d rows, want none", len(ws.created))
	}
}

// Files are moved before the row's transaction; when it fails they go back
// to the pending area, so the same submission can be sent again.
func TestSubmitFormReturnsFilesWhenTheRowIsNotWritten(t *testing.T) {
	a, ws, _ := submitApp(fileField("docs", false))
	root := useLocalProjectFiles(t, a)

	uploaded := uploadToForm(t, a, submitter, "report.pdf", "%PDF-1.4")

	ws.createErr = errors.New("deadlock")
	_, appErr := submit(a, submitter, model.FormValue{Name: "docs", Value: uploaded.ID})
	if appErr == nil || appErr.Status != http.StatusInternalServerError {
		t.Fatalf("want 500, got %v", appErr)
	}

	if content := readFile(t, filepath.Join(root, wsID, tbl1, "pending", uploaded.ID+".pdf")); content != "%PDF-1.4" {
		t.Errorf("pending file holds %q", content)
	}

	ws.createErr = nil
	if _, appErr := submit(a, submitter, model.FormValue{Name: "docs", Value: uploaded.ID}); appErr != nil {
		t.Fatalf("second SubmitForm: %v", appErr)
	}
	if len(ws.created) != 1 || len(ws.created[0].Attachments) != 1 {
		t.Errorf("created %+v, want one row with its file", ws.created)
	}
}
