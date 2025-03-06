// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type API struct {
	p      *Plugin
	router *mux.Router
}

func NewAPI(p *Plugin) *API {
	router := mux.NewRouter()

	api := &API{p: p, router: router}

	api.handleStaticFiles(router)

	// iFrame support
	router.HandleFunc("/iframe/mattermostTab", api.iFrame).Methods("GET")
	router.HandleFunc("/iframe-manifest", api.iFrameManifest).Methods("GET")

	return api
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// handleStaticFiles handles the static files under the assets directory.
func (a *API) handleStaticFiles(r *mux.Router) {
	bundlePath, err := a.p.API.GetBundlePath()
	if err != nil {
		a.p.API.LogWarn("Failed to get bundle path.", "error", err.Error())
		return
	}

	// This will serve static files from the 'assets' directory under '/static/<filename>'
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(bundlePath, "assets")))))
}

/*
func (a *API) mattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
*/
