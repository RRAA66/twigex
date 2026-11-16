// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/twigex/twigex/internal/parse"
	"github.com/twigex/twigex/model"
)

const (
	maxFormUploadBytes = 100 << 20
	maxFormImageBytes  = 6 << 20
)

func (a *API) initWorkspaceForms() {
	if !*a.app.ConfigStore.Config.FeatureFlags.Forms {
		return
	}

	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms", a.createForm).Methods("POST")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}", a.getForm).Methods("GET")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}", a.updateForm).Methods("PUT")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}", a.deleteForm).Methods("DELETE")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}/submissions", a.getFormSubmissions).Methods("GET")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}/images/{kind}", a.setFormImage).Methods("POST")
	a.BaseRoutes.Workspaces.HandleFunc("/{id}/table/{tid}/forms/{fid}/images/{kind}", a.removeFormImage).Methods("DELETE")
}

func (a *API) createForm(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var body model.WorkspaceFormRequest
	if !decodeBody(w, r, &body) {
		return
	}

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	form, appErr := a.app.CreateForm(r.Context(), params["id"], params["tid"], body.Form, body.Fields, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func (a *API) getForm(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	form, appErr := a.app.GetForm(r.Context(), params["id"], params["tid"], params["fid"], *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func (a *API) updateForm(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var body model.WorkspaceFormRequest
	if !decodeBody(w, r, &body) {
		return
	}

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	form, appErr := a.app.UpdateForm(r.Context(), params["id"], params["tid"], params["fid"], body.Form, body.Fields, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func (a *API) deleteForm(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	if appErr := a.app.DeleteForm(r.Context(), params["id"], params["tid"], params["fid"], *user); appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) initPublicForms() {
	if !*a.app.ConfigStore.Config.FeatureFlags.Forms {
		return
	}

	forms := a.BaseRoutes.PublicSharedFiles.PathPrefix("/forms").Subrouter()
	forms.HandleFunc("/{token}", a.getPublicForm).Methods("GET")
	forms.HandleFunc("/{token}/submit", a.submitPublicForm).Methods("POST")
	forms.HandleFunc("/{token}/files", a.uploadPublicFormFiles).Methods("POST")
	forms.HandleFunc("/{token}/linked/{tableId}", a.getPublicLinkedItems).Methods("GET")
	forms.Use(a.RequireCSRF)
}

func (a *API) getFormSubmissions(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	page := parse.Int(r.URL.Query().Get("page"), 1)

	submissions, appErr := a.app.GetFormSubmissions(r.Context(), params["id"], params["tid"], params["fid"], page, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, submissions)
}

func (a *API) getPublicForm(w http.ResponseWriter, r *http.Request) {
	user, _ := a.app.GetCurrentUser(r)

	form, appErr := a.app.GetPublicForm(r.Context(), mux.Vars(r)["token"], user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func (a *API) getPublicLinkedItems(w http.ResponseWriter, r *http.Request) {
	user, _ := a.app.GetCurrentUser(r)

	params := mux.Vars(r)
	query := r.URL.Query()
	page := parse.Int(query.Get("page"), 1)
	limit := parse.Int(query.Get("limit"), 0)

	items, appErr := a.app.GetPublicLinkedItems(r.Context(), params["token"], params["tableId"], query.Get("search"), page, limit, user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, items)
}

func (a *API) submitPublicForm(w http.ResponseWriter, r *http.Request) {
	var body model.FormSubmitRequest
	if !decodeBody(w, r, &body) {
		return
	}

	user, _ := a.app.GetCurrentUser(r)

	taskID, appErr := a.app.SubmitForm(r.Context(), mux.Vars(r)["token"], body, user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"id": taskID})
}

func (a *API) uploadPublicFormFiles(w http.ResponseWriter, r *http.Request) {
	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	form, appErr := a.app.CheckFormUpload(r.Context(), mux.Vars(r)["token"], user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	files, ok := parseUploadedFiles(w, r, maxFormUploadBytes, "files")
	if !ok {
		return
	}

	uploaded, appErr := a.app.UploadFormFiles(r.Context(), form, files, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, uploaded)
}

func (a *API) setFormImage(w http.ResponseWriter, r *http.Request) {
	files, ok := parseUploadedFiles(w, r, maxFormImageBytes, "file")
	if !ok {
		return
	}

	if len(files) != 1 {
		respondAppError(w, r, model.NewAppError("request.invalid", http.StatusBadRequest))
		return
	}

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	params := mux.Vars(r)

	form, appErr := a.app.SetFormImage(r.Context(), params["id"], params["tid"], params["fid"], params["kind"], files[0], *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func (a *API) removeFormImage(w http.ResponseWriter, r *http.Request) {
	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	params := mux.Vars(r)

	form, appErr := a.app.RemoveFormImage(r.Context(), params["id"], params["tid"], params["fid"], params["kind"], *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, form)
}

func parseUploadedFiles(w http.ResponseWriter, r *http.Request, maxBytes int64, key string) ([]*multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondAppError(w, r, model.NewAppError("workspace.form_file_too_large", http.StatusRequestEntityTooLarge))
			return nil, false
		}

		respondAppError(w, r, model.NewAppError("request.invalid", http.StatusBadRequest))
		return nil, false
	}

	return r.MultipartForm.File[key], true
}
