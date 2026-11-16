// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"slices"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/twigex/twigex/app"
	"github.com/twigex/twigex/config"
	"github.com/twigex/twigex/model"
)

func formRoutes(t *testing.T, formsOn bool) []string {
	t.Helper()

	a := &API{
		app: &app.App{ConfigStore: config.ConfigStore{Config: &model.ServerConfig{
			FeatureFlags: model.FeatureFlags{Forms: model.NewBool(formsOn)},
		}}},
		BaseRoutes: &Routes{Workspaces: mux.NewRouter(), PublicSharedFiles: mux.NewRouter()},
	}
	a.initWorkspaces()
	a.initPublicForms()

	return append(routesUnder(t, a.BaseRoutes.Workspaces), routesUnder(t, a.BaseRoutes.PublicSharedFiles)...)
}

func routesUnder(t *testing.T, router *mux.Router) []string {
	t.Helper()

	var routes []string
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		if strings.Contains(path, "/forms") && len(methods) > 0 {
			routes = append(routes, strings.Join(methods, ",")+" "+path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	slices.Sort(routes)

	return routes
}

func TestFormRoutesAreNotRegisteredWithFormsOff(t *testing.T) {
	if routes := formRoutes(t, false); len(routes) != 0 {
		t.Fatalf("want no form routes, got %v", routes)
	}
}

func TestFormRoutesAreRegisteredWithFormsOn(t *testing.T) {
	want := []string{
		"DELETE /{id}/table/{tid}/forms/{fid}",
		"GET /{id}/table/{tid}/forms/{fid}",
		"GET /{id}/table/{tid}/forms/{fid}/submissions",
		"POST /{id}/table/{tid}/forms",
		"PUT /{id}/table/{tid}/forms/{fid}",
		"GET /forms/{token}",
		"GET /forms/{token}/linked/{tableId}",
		"POST /forms/{token}/submit",
	}

	if routes := formRoutes(t, true); !slices.Equal(routes, want) {
		t.Fatalf("want %v, got %v", want, routes)
	}
}
