// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/twigex/twigex/app"
	"github.com/twigex/twigex/internal/i18n"
	"github.com/twigex/twigex/store"
)

type noSessionStore struct {
	store.AuthStore
}

func (noSessionStore) GetUserID(*http.Request) (*string, error) {
	return nil, errors.New("no session")
}

type readCounter struct {
	io.Reader
	read int
}

func (r *readCounter) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n

	return n, err
}

func TestUploadPublicFormFilesRefusesAGuestBeforeReadingTheFiles(t *testing.T) {
	if err := i18n.Init("../i18n"); err != nil {
		t.Fatalf("i18n: %v", err)
	}

	a := &API{app: &app.App{Store: store.Store{Auth: noSessionStore{}}}}

	router := mux.NewRouter()
	router.HandleFunc("/{token}/files", a.uploadPublicFormFiles)

	body := &readCounter{Reader: strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"files\"; filename=\"a.pdf\"\r\n\r\n%PDF\r\n--x--\r\n")}
	r := httptest.NewRequest(http.MethodPost, "/token-1/files", body)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=x")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", rec.Code)
	}
	if body.read != 0 {
		t.Errorf("read %d bytes of the body, want none", body.read)
	}
}
