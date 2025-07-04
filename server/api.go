// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
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
	router.HandleFunc("/iframe-manifest", api.adminRequired(api.appManifest)).Methods(http.MethodGet)
	router.HandleFunc("/csp-report", api.cspReport).Methods(http.MethodPost)

	// Icon upload and retrieval endpoints
	router.HandleFunc("/icons/upload", api.adminRequired(api.uploadIcon)).Methods(http.MethodPost)
	router.HandleFunc("/icons/{iconType}", api.adminRequired(api.getIcon)).Methods(http.MethodGet)
	router.HandleFunc("/icons/{iconType}", api.adminRequired(api.deleteIcon)).Methods(http.MethodDelete)
	router.HandleFunc("/icons/{iconType}/exists", api.adminRequired(api.iconExists)).Methods(http.MethodGet)

	// Embedded SSO login
	api.serveSSO(router)

	return api
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// adminRequired is a middleware that checks if the user is an admin
func (a *API) adminRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// User must be logged in
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		// User must be an admin
		if !a.p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Credentials are valid, call the next handler
		next(w, r)
	}
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
