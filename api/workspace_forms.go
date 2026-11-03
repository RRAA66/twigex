// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/twigex/twigex/model"
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
