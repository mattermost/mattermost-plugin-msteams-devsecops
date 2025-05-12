// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
	router.HandleFunc("/iframe/mattermostTab", api.iFrame).Methods(http.MethodGet)
	router.HandleFunc("/iframe/authenticate", api.authenticate).Methods(http.MethodGet)
	router.HandleFunc("/iframe/notification_preview", api.iframeNotificationPreview).Methods(http.MethodGet)
	router.HandleFunc("/iframe-manifest", api.appManifest).Methods(http.MethodGet)
	router.HandleFunc("/csp-report", api.cspReport).Methods(http.MethodPost)

	// Embedded SSO login
	api.serveSSO(router)

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

// handleErrorWithCode logs the internal error and sends the public facing error
// message as JSON in a response with the provided code.
func handleErrorWithCode(logger logrus.FieldLogger, w http.ResponseWriter, code int, publicErrorMsg string, internalErr error) {
	if internalErr != nil {
		logger = logger.WithError(internalErr)
	}

	if code >= http.StatusInternalServerError {
		logger.Error(publicErrorMsg)
	} else {
		logger.Warn(publicErrorMsg)
	}

	handleResponseWithCode(w, code, publicErrorMsg)
}

// handleResponseWithCode logs the internal error and sends the public facing error
// message as JSON in a response with the provided code.
func handleResponseWithCode(w http.ResponseWriter, code int, publicMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	responseMsg, _ := json.Marshal(struct {
		Error string `json:"error"` // A public facing message providing details about the error.
	}{
		Error: publicMsg,
	})
	_, _ = w.Write(responseMsg)
}
