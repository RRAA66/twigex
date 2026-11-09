// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/twigex/twigex/internal/parse"
	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

const (
	maxFormUploadBytes = 100 << 20
	maxFormImageBytes  = 6 << 20
)

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

	query := r.URL.Query()
	page := parse.Int(query.Get("page"), 1)

	submissions, appErr := a.app.GetFormSubmissions(r.Context(), params["id"], params["tid"], params["fid"], submissionsFilter(query), page, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	respondJSON(w, http.StatusOK, submissions)
}

func submissionsFilter(query url.Values) model.FormSubmissionsFilter {
	return model.FormSubmissionsFilter{
		From: int64(parse.Int(query.Get("from"), 0)),
		To:   int64(parse.Int(query.Get("to"), 0)),
	}
}

func (a *API) exportFormSubmissions(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user, appErr := a.app.GetCurrentUser(r)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	query := r.URL.Query()

	var labels map[string]string
	if raw := query.Get("labels"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &labels); err != nil {
			respondAppError(w, r, model.NewAppError("request.invalid", http.StatusBadRequest))
			return
		}
	}

	export, appErr := a.app.PrepareFormSubmissionsExport(r.Context(), params["id"], params["tid"], params["fid"], submissionsFilter(query), query.Get("tz"), query.Get("clock"), labels, *user)
	if appErr != nil {
		respondAppError(w, r, appErr)
		return
	}

	name := export.FileName()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, sanitizeHeaderFilename(name), url.PathEscape(name)))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	if err := export.WriteCSV(r.Context(), w, flush); err != nil {
		tlog.Errorw("Failed to export form submissions", "form_id", params["fid"], "user_id", user.ID, "error", err)
	}
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

	taskID, appErr := a.app.SubmitForm(r.Context(), mux.Vars(r)["token"], body, user, a.app.ClientIP(r))
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

	form, appErr := a.app.CheckFormUpload(r.Context(), mux.Vars(r)["token"], user, a.app.ClientIP(r))
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
