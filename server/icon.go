// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/store/pluginstore"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/sirupsen/logrus"
)

type IconType string

const (
	IconTypeColor   IconType = "color"
	IconTypeOutline IconType = "outline"
)

// IsValid checks if the IconType is valid
func (it IconType) IsValid() bool {
	return it == IconTypeColor || it == IconTypeOutline
}

// uploadIcon handles file uploads for custom icons
func (a *API) uploadIcon(w http.ResponseWriter, r *http.Request) {
	logger := logrus.StandardLogger()

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

	// Parse multipart form
	err := r.ParseMultipartForm(1 << 20) // 1MB max
	if err != nil {
		handleErrorWithCode(logger, w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	// Get the icon type from form data
	iconTypeStr := r.FormValue("iconType")
	iconType := IconType(iconTypeStr)
	if !iconType.IsValid() {
		http.Error(w, "Invalid icon type. Must be 'color' or 'outline'", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("icon")
	if err != nil {
		handleErrorWithCode(logger, w, http.StatusBadRequest, "Failed to get uploaded file", err)
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/png") {
		http.Error(w, "Invalid file type. Must be PNG", http.StatusBadRequest)
		return
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		handleErrorWithCode(logger, w, http.StatusInternalServerError, "Failed to read file data", err)
		return
	}

	// Validate file size
	if len(data) > 1024*1024 { // 1MB
		http.Error(w, "File too large. Maximum size is 1MB", http.StatusBadRequest)
		return
	}

	// Validate image dimensions
	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "Invalid PNG image", http.StatusBadRequest)
		return
	}

	// Check if image is square and within acceptable size range
	if img.Width != img.Height {
		http.Error(w, "Image must be square (width and height must be equal)", http.StatusBadRequest)
		return
	}

	if img.Width < 150 || img.Width > 300 || img.Height < 150 || img.Height > 300 {
		http.Error(w, "Image must be between 150x150 and 300x300 pixels, and should be 192x192 pixels", http.StatusBadRequest)
		return
	}

	// Store the icon in KV store
	err = a.p.pluginStore.StoreIcon(string(iconType), data)
	if err != nil {
		handleErrorWithCode(logger, w, http.StatusInternalServerError, "Failed to store icon", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"success":  true,
		"iconPath": fmt.Sprintf("/plugins/com.mattermost.plugin-msteams-devsecops/icons/%s", iconType),
	}
	_ = json.NewEncoder(w).Encode(response)
}

// getIcon serves custom icons or falls back to default icons
func (a *API) getIcon(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iconTypeStr := vars["iconType"]
	iconType := IconType(iconTypeStr)

	if !iconType.IsValid() {
		http.Error(w, "Invalid icon type", http.StatusBadRequest)
		return
	}

	// Try to get custom icon from KV store
	iconData, err := a.p.pluginStore.GetIcon(string(iconType))
	if err != nil {
		// If not found, serve default icon
		var defaultData []byte
		if iconType == IconTypeColor {
			defaultData = assets.LogoColorData
		} else {
			defaultData = assets.LogoOutlineData
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(defaultData)
		return
	}

	// Serve custom icon
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(iconData)
}

// deleteIcon removes a custom icon and reverts to default
func (a *API) deleteIcon(w http.ResponseWriter, r *http.Request) {
	logger := logrus.StandardLogger()

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

	vars := mux.Vars(r)
	iconTypeStr := vars["iconType"]
	iconType := IconType(iconTypeStr)

	if !iconType.IsValid() {
		http.Error(w, "Invalid icon type", http.StatusBadRequest)
		return
	}

	// Delete from KV store
	err := a.p.pluginStore.DeleteIcon(string(iconType))
	if err != nil {
		// If the icon doesn't exist, that's fine
		var notFoundErr *pluginstore.ErrNotFound
		if !errors.Is(err, notFoundErr) {
			handleErrorWithCode(logger, w, http.StatusInternalServerError, "Failed to delete icon", err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"success": true,
		"message": "Icon deleted successfully",
	}
	_ = json.NewEncoder(w).Encode(response)
}

// iconExists checks if a custom icon exists in the KV store
func (a *API) iconExists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iconTypeStr := vars["iconType"]
	iconType := IconType(iconTypeStr)

	if !iconType.IsValid() {
		http.Error(w, "Invalid icon type", http.StatusBadRequest)
		return
	}

	// Check if custom icon exists in KV store
	_, err := a.p.pluginStore.GetIcon(string(iconType))
	exists := err == nil

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"exists": exists,
	}
	_ = json.NewEncoder(w).Encode(response)
}
