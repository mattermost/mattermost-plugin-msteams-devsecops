// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
)

// serveSSO configures routes for handling SSO endpoints
func (a *API) serveSSO(router *mux.Router) {
	router.HandleFunc("/sso/wait", a.handleSSOWait).Methods(http.MethodGet)
	router.HandleFunc("/sso/complete", a.handleSSOComplete).Methods(http.MethodGet)
}

// handleSSOWait handles the SSO waiting page that listens for messages from the popup window
func (a *API) handleSSOWait(w http.ResponseWriter, r *http.Request) {
	iFrameCtx, err := a.createIFrameContext("", nil)
	if err != nil {
		a.p.API.LogError("Failed to create SSO wait context", "error", err.Error())
		http.Error(w, "Failed to create SSO wait context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.SSOWaitHTMLTemplate, iFrameCtx)
	if err != nil {
		a.p.API.LogError("Failed to format SSO wait HTML", "error", err.Error())
		http.Error(w, "Failed to format SSO wait HTML", http.StatusInternalServerError)
		return
	}

	a.returnCSPHeaders(w, iFrameCtx)
	w.Header().Set("Content-Type", "text/html")

	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogWarn("Unable to serve the SSO wait page", "error", err.Error())
	}
}

// handleSSOComplete handles the SSO completion page that posts messages back to the parent window
func (a *API) handleSSOComplete(w http.ResponseWriter, r *http.Request) {
	iFrameCtx, err := a.createIFrameContext("", nil)
	if err != nil {
		a.p.API.LogError("Failed to create SSO complete context", "error", err.Error())
		http.Error(w, "Failed to create SSO complete context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.SSOCompleteHTMLTemplate, iFrameCtx)
	if err != nil {
		a.p.API.LogError("Failed to format SSO complete HTML", "error", err.Error())
		http.Error(w, "Failed to format SSO complete HTML", http.StatusInternalServerError)
		return
	}

	a.returnCSPHeaders(w, iFrameCtx)
	w.Header().Set("Content-Type", "text/html")

	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogWarn("Unable to serve the SSO complete page", "error", err.Error())
	}
}
