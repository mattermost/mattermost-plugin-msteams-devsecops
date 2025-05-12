// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"strings"

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
	iframeCtx, err := a.createIFrameContext("", nil)
	if err != nil {
		a.p.API.LogError("Failed to create SSO wait context", "error", err.Error())
		http.Error(w, "Failed to create SSO wait context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.SSOWaitHTMLTemplate, iframeCtx)
	if err != nil {
		a.p.API.LogError("Failed to format SSO wait HTML", "error", err.Error())
		http.Error(w, "Failed to format SSO wait HTML", http.StatusInternalServerError)
		return
	}

	cspDirectives := []string{
		"default-src 'none'",
		"style-src 'self' 'unsafe-inline'",
		"script-src 'nonce-" + iframeCtx.Nonce + "'",
		"connect-src 'self'",
		"img-src 'self'",
	}

	w.Header().Set("Content-Security-Policy", strings.Join(cspDirectives, "; "))
	w.Header().Set("Content-Type", "text/html")

	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogWarn("Unable to serve the SSO wait page", "error", err.Error())
	}
}

// handleSSOComplete handles the SSO completion page that posts messages back to the parent window
func (a *API) handleSSOComplete(w http.ResponseWriter, r *http.Request) {
	iframeCtx, err := a.createIFrameContext("", nil)
	if err != nil {
		a.p.API.LogError("Failed to create SSO complete context", "error", err.Error())
		http.Error(w, "Failed to create SSO complete context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.SSOCompleteHTMLTemplate, iframeCtx)
	if err != nil {
		a.p.API.LogError("Failed to format SSO complete HTML", "error", err.Error())
		http.Error(w, "Failed to format SSO complete HTML", http.StatusInternalServerError)
		return
	}

	cspDirectives := []string{
		"default-src 'none'",
		"style-src 'self' 'unsafe-inline'",
		"script-src https://res.cdn.office.net 'nonce-" + iframeCtx.Nonce + "'",
		"connect-src https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net",
		"img-src 'self'",
		"frame-ancestors *", // Allow being embedded in iframes
	}

	w.Header().Set("Content-Security-Policy", strings.Join(cspDirectives, "; "))
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("X-Frame-Options", "ALLOWALL") // Allow iframe embedding

	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogWarn("Unable to serve the SSO complete page", "error", err.Error())
	}
}
