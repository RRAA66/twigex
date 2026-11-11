// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/twigex/twigex/model"
)

const pngHeader = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

type fakeFormImageStore struct {
	*fakeFormStore
	attachments map[string]model.WorkspaceAttachment
}

func (f *fakeFormImageStore) GetAttachment(fileID, workspaceID string) (*model.WorkspaceAttachment, error) {
	attachment, ok := f.attachments[fileID]
	if !ok || attachment.WorkspaceID != workspaceID {
		return nil, nil
	}

	return &attachment, nil
}

func (f *fakeFormImageStore) SetFormImage(_ context.Context, formID, kind string, image *model.WorkspaceAttachment, replacedID string) error {
	f.writes++

	id := ""
	if image != nil {
		f.attachments[image.ID] = *image
		id = image.ID
	}

	if kind == model.FormImageCover {
		f.forms[formID].CoverImage = id
	} else {
		f.forms[formID].Logo = id
	}

	delete(f.attachments, replacedID)
	return nil
}

func (f *fakeFormImageStore) DeleteTableView(_ context.Context, workspaceID, tableID, viewID string) (bool, error) {
	form, ok := f.forms[viewID]
	if ok && form.WorkspaceID == workspaceID && form.TableID == tableID {
		delete(f.forms, viewID)
		delete(f.attachments, form.CoverImage)
		delete(f.attachments, form.Logo)
	}

	return true, nil
}

func imageApp(t *testing.T) (*App, *fakeFormImageStore, string) {
	t.Helper()

	a, forms := formApp(workspaceAdminAccess(), true)
	ws := &fakeFormImageStore{fakeFormStore: forms, attachments: map[string]model.WorkspaceAttachment{}}
	a.Store.Workspace = ws

	return a, ws, useLocalProjectFiles(t, a)
}

func setImage(t *testing.T, a *App, kind, name, content string) (*model.WorkspaceFormWithFields, *model.AppError) {
	t.Helper()

	image := uploadedFiles(t, "file", map[string]string{name: content})[0]
	return a.SetFormImage(context.Background(), wsID, tbl1, "form-1", kind, image, workspaceAdmin)
}

func TestSetFormImageStoresItAsAnAttachmentOfTheForm(t *testing.T) {
	a, ws, root := imageApp(t)

	form, appErr := setImage(t, a, model.FormImageCover, "Holiday.PNG", pngHeader)
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	attachment, ok := ws.attachments[form.CoverImage]
	if !ok {
		t.Fatalf("form's cover %q is not an attachment", form.CoverImage)
	}
	if attachment.WorkspaceID != wsID || attachment.TableID != tbl1 || attachment.TaskID != "form-1" ||
		attachment.UserID != workspaceAdmin.ID || attachment.MimeType != "image/png" || attachment.Name != "cover.png" {
		t.Errorf("stored %+v, want form-1's png cover by %s", attachment, workspaceAdmin.ID)
	}

	if content := readFile(t, filepath.Join(root, wsID, tbl1, "form-1", attachment.ID+".png")); content != pngHeader {
		t.Errorf("cover file holds %q", content)
	}
}

func TestSetFormImageRefusesWhatIsNotAnImage(t *testing.T) {
	cases := map[string]string{
		"page.html":   "<html><script>alert(1)</script></html>",
		"fake.png":    "<html><script>alert(1)</script></html>",
		"drawing.svg": "<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			a, ws, root := imageApp(t)

			_, appErr := setImage(t, a, model.FormImageLogo, name, content)
			if appErr == nil || appErr.Status != http.StatusBadRequest {
				t.Fatalf("want 400, got %v", appErr)
			}
			if ws.writes != 0 {
				t.Errorf("wrote %d times, want none", ws.writes)
			}
			if _, err := os.Stat(root); !os.IsNotExist(err) {
				t.Errorf("wrote a file: %v", err)
			}
		})
	}
}

func TestSetFormImageRefusesAnUnknownKind(t *testing.T) {
	a, ws, _ := imageApp(t)

	_, appErr := setImage(t, a, "background", "a.png", pngHeader)
	if appErr == nil || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want 400, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestSetFormImageOnlyOnTheFormsWorkspaceAndTable(t *testing.T) {
	cases := []struct {
		name, workspace, table string
		want                   int
	}{
		{"other table", wsID, tbl2, http.StatusNotFound},
		{"other workspace", "workspace-2", tbl1, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, ws, _ := imageApp(t)
			ws.onlyMember = ""

			image := uploadedFiles(t, "file", map[string]string{"a.png": pngHeader})[0]
			_, appErr := a.SetFormImage(context.Background(), tc.workspace, tc.table, "form-1", model.FormImageCover, image, workspaceAdmin)
			if appErr == nil || appErr.Status != tc.want {
				t.Fatalf("want %d, got %v", tc.want, appErr)
			}
			if ws.writes != 0 {
				t.Errorf("wrote %d times, want none", ws.writes)
			}
		})
	}
}

func TestSetFormImageIsOpenToWhoManagesTheTablesViews(t *testing.T) {
	a, ws, _ := imageApp(t)
	ws.fakeAccessWorkspaceStore = memberWithoutViewPermissions()
	a.WorkspaceRoles = formRoles{viewsTable: tbl1}

	image := uploadedFiles(t, "file", map[string]string{"a.png": pngHeader})[0]
	if _, appErr := a.SetFormImage(context.Background(), wsID, tbl1, "form-1", model.FormImageCover, image, accessUser()); appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	if _, appErr := a.RemoveFormImage(context.Background(), wsID, tbl1, "form-1", model.FormImageCover, accessUser()); appErr != nil {
		t.Fatalf("RemoveFormImage: %v", appErr)
	}
}

func TestSetFormImageNeedsThePermissionToChangeViews(t *testing.T) {
	a, ws, _ := imageApp(t)
	ws.fakeAccessWorkspaceStore = memberWithoutViewPermissions()
	a.WorkspaceRoles = formRoles{}

	image := uploadedFiles(t, "file", map[string]string{"a.png": pngHeader})[0]
	_, appErr := a.SetFormImage(context.Background(), wsID, tbl1, "form-1", model.FormImageCover, image, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("want 403, got %v", appErr)
	}

	_, appErr = a.RemoveFormImage(context.Background(), wsID, tbl1, "form-1", model.FormImageCover, accessUser())
	if appErr == nil || appErr.Status != http.StatusForbidden {
		t.Fatalf("remove: want 403, got %v", appErr)
	}
	if ws.writes != 0 {
		t.Errorf("wrote %d times, want none", ws.writes)
	}
}

func TestSetFormImageReplacesTheOldOne(t *testing.T) {
	a, ws, root := imageApp(t)

	first, appErr := setImage(t, a, model.FormImageLogo, "old.png", pngHeader)
	if appErr != nil {
		t.Fatalf("first SetFormImage: %v", appErr)
	}

	second, appErr := setImage(t, a, model.FormImageLogo, "new.png", pngHeader)
	if appErr != nil {
		t.Fatalf("second SetFormImage: %v", appErr)
	}

	if second.Logo == first.Logo {
		t.Fatalf("logo stayed %s", second.Logo)
	}
	if _, ok := ws.attachments[first.Logo]; ok {
		t.Errorf("old logo %s is still an attachment", first.Logo)
	}

	formDir := filepath.Join(root, wsID, tbl1, "form-1")
	entries, err := os.ReadDir(formDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != second.Logo+".png" {
		t.Errorf("form's files are %v (%v), want only the new logo", entries, err)
	}
}

func TestRemoveFormImageDeletesItsFile(t *testing.T) {
	a, ws, root := imageApp(t)

	set, appErr := setImage(t, a, model.FormImageCover, "a.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF")
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	removed, appErr := a.RemoveFormImage(context.Background(), wsID, tbl1, "form-1", model.FormImageCover, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("RemoveFormImage: %v", appErr)
	}

	if removed.CoverImage != "" || len(ws.attachments) != 0 {
		t.Errorf("cover %q with attachments %v, want none", removed.CoverImage, ws.attachments)
	}
	if _, err := os.Stat(filepath.Join(root, wsID, tbl1, "form-1", set.CoverImage+".jpg")); !os.IsNotExist(err) {
		t.Errorf("cover file is still there: %v", err)
	}
}

// A form's image fields come from its builder only through SetFormImage: a
// client cannot point them at someone else's attachment.
func TestFormImagesAreNotTakenFromTheBuilder(t *testing.T) {
	a, ws, _ := imageApp(t)

	created, appErr := a.CreateForm(context.Background(), wsID, tbl1,
		model.WorkspaceForm{Name: "Copy", CoverImage: "row-file", Logo: "row-file"}, nil, workspaceAdmin)
	if appErr != nil {
		t.Fatalf("CreateForm: %v", appErr)
	}

	stored := ws.forms[created.ID]
	if stored.CoverImage != "" || stored.Logo != "" {
		t.Errorf("stored cover %q and logo %q, want none", stored.CoverImage, stored.Logo)
	}
}

func TestDeleteFormRemovesItsImages(t *testing.T) {
	a, _, root := imageApp(t)

	set, appErr := setImage(t, a, model.FormImageCover, "a.png", pngHeader)
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	if appErr := a.DeleteForm(context.Background(), wsID, tbl1, "form-1", workspaceAdmin); appErr != nil {
		t.Fatalf("DeleteForm: %v", appErr)
	}

	if _, err := os.Stat(filepath.Join(root, wsID, tbl1, "form-1", set.CoverImage+".png")); !os.IsNotExist(err) {
		t.Errorf("cover file is still there: %v", err)
	}
}

func TestDeletingAFormAsAViewRemovesItsImages(t *testing.T) {
	a, ws, root := imageApp(t)

	set, appErr := setImage(t, a, model.FormImageLogo, "a.png", pngHeader)
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	if _, appErr := a.DeleteWorkspaceTableView(context.Background(), wsID, tbl1, "form-1", workspaceAdmin); appErr != nil {
		t.Fatalf("DeleteWorkspaceTableView: %v", appErr)
	}

	if _, ok := ws.attachments[set.Logo]; ok {
		t.Error("logo's attachment is still there")
	}
	if _, err := os.Stat(filepath.Join(root, wsID, tbl1, "form-1", set.Logo+".png")); !os.IsNotExist(err) {
		t.Errorf("logo file is still there: %v", err)
	}
}

// fakeFormImageReaders has the admin and "assigned" as direct members and
// "grouped" as a member through a group only. Every row is someone else's.
type fakeFormImageReaders struct {
	*fakeFormImageStore
	privateView bool
	formErr     error
}

func (f *fakeFormImageReaders) GetForm(ctx context.Context, formID string) (*model.WorkspaceFormWithFields, error) {
	if f.formErr != nil {
		return nil, f.formErr
	}

	return f.fakeFormImageStore.GetForm(ctx, formID)
}

func (f *fakeFormImageReaders) IsMember(_, userID string) (bool, error) {
	return userID == workspaceAdmin.ID || userID == "assigned", nil
}

func (f *fakeFormImageReaders) UserHasAnyGroupAccess(_, userID string) (bool, error) {
	return userID == "grouped", nil
}

func (f *fakeFormImageReaders) GetTaskAssigneeID(string, string) (string, error) {
	return "someone-else", nil
}

func (f *fakeFormImageReaders) IsTableSingleSelect(string) (bool, error) {
	return false, nil
}

func (f *fakeFormImageReaders) GetViewSharedUserIDs(string) ([]string, error) {
	return nil, nil
}

func (f *fakeFormImageReaders) GetViewVisibility(context.Context, string) (bool, string, error) {
	return !f.privateView, "creator", nil
}

func TestFormImagesAreShownToWhoeverMayOpenTheForm(t *testing.T) {
	a, ws, _ := imageApp(t)

	cover, appErr := setImage(t, a, model.FormImageCover, "a.png", pngHeader)
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	ws.attachments["row-file"] = model.WorkspaceAttachment{ID: "row-file", WorkspaceID: wsID, TableID: tbl1, FieldID: "field-docs", TaskID: "task-1"}
	ws.attachments["forged"] = model.WorkspaceAttachment{ID: "forged", WorkspaceID: wsID, TableID: tbl1, FieldID: "form_cover", TaskID: "task-1"}

	readers := &fakeFormImageReaders{fakeFormImageStore: ws}
	a.Store.Workspace = readers
	a.WorkspaceRoles = assignedOnlyRoles{tables: map[string]bool{tbl1: true}}

	for _, user := range []model.User{{ID: "grouped"}, {ID: "assigned"}} {
		if _, appErr := a.GetWorkspaceAttachment(context.Background(), cover.CoverImage, wsID, user); appErr != nil {
			t.Errorf("%s: cover refused with %v", user.ID, appErr)
		}

		for _, other := range []string{"row-file", "forged"} {
			if _, appErr := a.GetWorkspaceAttachment(context.Background(), other, wsID, user); appErr == nil || appErr.Status != http.StatusForbidden {
				t.Errorf("%s: %s answered %v, want 403", user.ID, other, appErr)
			}
		}
	}

	if _, appErr := a.GetWorkspaceAttachment(context.Background(), cover.CoverImage, wsID, *outsider); appErr == nil || appErr.Status != http.StatusForbidden {
		t.Errorf("outsider: cover answered %v, want 403", appErr)
	}

	readers.privateView = true
	if _, appErr := a.GetWorkspaceAttachment(context.Background(), cover.CoverImage, wsID, model.User{ID: "grouped"}); appErr == nil {
		t.Error("cover of a form in a private view was shown to a member it is not shared with")
	}
}

func TestFormImageIsRefusedWhenItsFormCannotBeRead(t *testing.T) {
	a, ws, _ := imageApp(t)

	cover, appErr := setImage(t, a, model.FormImageCover, "a.png", pngHeader)
	if appErr != nil {
		t.Fatalf("SetFormImage: %v", appErr)
	}

	a.Store.Workspace = &fakeFormImageReaders{fakeFormImageStore: ws, formErr: errors.New("connection lost")}
	a.WorkspaceRoles = assignedOnlyRoles{tables: map[string]bool{tbl1: true}}

	_, appErr = a.GetWorkspaceAttachment(context.Background(), cover.CoverImage, wsID, model.User{ID: "assigned"})
	if appErr == nil || appErr.Status != http.StatusInternalServerError {
		t.Errorf("want 500, got %v", appErr)
	}
}
