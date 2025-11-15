// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	_ "embed"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Embed test PNG files for dimension validation tests
var (
	//go:embed test/test-150x150.png
	testPNG150x150 []byte

	//go:embed test/test-300x300.png
	testPNG300x300 []byte

	//go:embed test/test-non-square.png
	testPNGNonSquare []byte

	//go:embed test/test-too-big.png
	testPNGTooBig []byte

	//go:embed test/test-too-small.png
	testPNGTooSmall []byte
)

// loadTestPNG loads a real PNG file from the assets directory for testing
func loadTestPNG() []byte {
	// Use the assets.go embedded data instead of trying to read files from disk
	// This is more reliable since the assets are embedded in the binary
	return assets.LogoColorData
}

// createTestPNGData returns test PNG data for various test scenarios
func createTestPNGData(width, height int) []byte {
	switch {
	case width == 192 && height == 192:
		// Use the default logo for valid 192x192 dimensions
		return loadTestPNG()
	case width == 150 && height == 150:
		return testPNG150x150
	case width == 300 && height == 300:
		return testPNG300x300
	case width == 100 && height == 100:
		return testPNGTooSmall
	case width == 400 && height == 400:
		return testPNGTooBig
	case width == 200 && height == 150:
		return testPNGNonSquare
	default:
		// Fallback to the default logo for unspecified dimensions
		return loadTestPNG()
	}
}

// createMultipartRequest creates a multipart form request for icon upload
func createMultipartRequest(iconType string, imageData []byte, contentType string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add iconType field
	if iconType != "" {
		err := writer.WriteField("iconType", iconType)
		if err != nil {
			return nil, err
		}
	}

	// Add file field
	if imageData != nil {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="icon"; filename="test-icon.png"`)
		if contentType == "" {
			h.Set("Content-Type", "image/png")
		} else {
			h.Set("Content-Type", contentType)
		}
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, err
		}
		_, err = part.Write(imageData)
		if err != nil {
			return nil, err
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/icons/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

func TestIconType(t *testing.T) {
	t.Run("IsValid returns true for valid icon types", func(t *testing.T) {
		assert.True(t, IconTypeColor.IsValid())
		assert.True(t, IconTypeOutline.IsValid())
	})

	t.Run("IsValid returns false for invalid icon types", func(t *testing.T) {
		assert.False(t, IconType("invalid").IsValid())
		assert.False(t, IconType("").IsValid())
		assert.False(t, IconType("COLOR").IsValid()) // case sensitive
	})
}

func TestUploadIcon(t *testing.T) {
	th := setupTestHelper(t)

	// Create admin user
	team := th.SetupTeam(t)
	admin := th.SetupUser(t, team)

	// Grant admin permissions
	_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
	require.Nil(t, appErr)

	// Create regular user
	regularUser := th.SetupUser(t, team)

	validPNGData := createTestPNGData(192, 192)

	t.Run("successful upload with valid PNG", func(t *testing.T) {
		req, err := createMultipartRequest("color", validPNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"success":true`)
		assert.Contains(t, string(body), "/plugins/com.mattermost.plugin-msteams-devsecops/icons/color")

		// Verify the icon was stored correctly in the KV store
		storedIconData, err := th.p.pluginStore.GetIcon("color")
		require.NoError(t, err, "Should be able to retrieve stored icon")
		assert.Equal(t, validPNGData, storedIconData, "Stored icon data should match uploaded data")
	})

	t.Run("successful upload with outline icon type", func(t *testing.T) {
		req, err := createMultipartRequest("outline", validPNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(t, string(body), `"success":true`)
		assert.Contains(t, string(body), "/plugins/com.mattermost.plugin-msteams-devsecops/icons/outline")

		// Verify the icon was stored correctly in the KV store
		storedIconData, err := th.p.pluginStore.GetIcon("outline")
		require.NoError(t, err, "Should be able to retrieve stored icon")
		assert.Equal(t, validPNGData, storedIconData, "Stored icon data should match uploaded data")
	})

	t.Run("unauthorized when no user ID provided", func(t *testing.T) {
		req, err := createMultipartRequest("color", validPNGData, "")
		require.NoError(t, err)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("forbidden when user is not admin", func(t *testing.T) {
		req, err := createMultipartRequest("color", validPNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", regularUser.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("bad request when multipart form is invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/icons/upload", strings.NewReader("invalid form data"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("bad request when icon type is invalid", func(t *testing.T) {
		req, err := createMultipartRequest("invalid", validPNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Invalid icon type")
	})

	t.Run("bad request when icon type is missing", func(t *testing.T) {
		req, err := createMultipartRequest("", validPNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("bad request when file is missing", func(t *testing.T) {
		req, err := createMultipartRequest("color", nil, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Failed to get uploaded file")
	})

	t.Run("bad request when file type is not PNG", func(t *testing.T) {
		jpegData := []byte("fake jpeg data")
		req, err := createMultipartRequest("color", jpegData, "image/jpeg")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyContent, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(bodyContent), "Invalid file type")
	})

	t.Run("bad request when file is too large", func(t *testing.T) {
		// Create a file larger than MaxUploadSize by padding the real PNG
		basePNG := loadTestPNG()
		largePNGData := make([]byte, MaxUploadSize+1)
		copy(largePNGData, basePNG)

		req, err := createMultipartRequest("color", largePNGData, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "File too large")
	})

	t.Run("bad request when image is not square", func(t *testing.T) {
		// Create a non-square PNG (width != height)
		nonSquarePNG := createTestPNGData(200, 150)

		req, err := createMultipartRequest("color", nonSquarePNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Image must be square")
	})

	t.Run("bad request when image dimensions are too small", func(t *testing.T) {
		smallPNG := createTestPNGData(100, 100)

		req, err := createMultipartRequest("color", smallPNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Image must be between 150x150 and 300x300 pixels")
	})

	t.Run("bad request when image dimensions are too large", func(t *testing.T) {
		largePNG := createTestPNGData(400, 400)

		req, err := createMultipartRequest("color", largePNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Image must be between 150x150 and 300x300 pixels")
	})

	t.Run("bad request when PNG is invalid", func(t *testing.T) {
		invalidPNG := []byte("not a png file")

		req, err := createMultipartRequest("color", invalidPNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Invalid PNG image")
	})

	t.Run("successful upload with minimum valid dimensions", func(t *testing.T) {
		// Use the 150x150 PNG (minimum valid size)
		minValidPNG := createTestPNGData(150, 150)

		req, err := createMultipartRequest("color", minValidPNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"success":true`)

		// Verify the icon was stored correctly in the KV store
		storedIconData, err := th.p.pluginStore.GetIcon("color")
		require.NoError(t, err, "Should be able to retrieve stored icon")
		assert.Equal(t, minValidPNG, storedIconData, "Stored icon data should match uploaded data")
	})

	t.Run("successful upload with maximum valid dimensions", func(t *testing.T) {
		// Use the 300x300 PNG (maximum valid size)
		maxValidPNG := createTestPNGData(300, 300)

		req, err := createMultipartRequest("color", maxValidPNG, "")
		require.NoError(t, err)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"success":true`)

		// Verify the icon was stored correctly in the KV store
		storedIconData, err := th.p.pluginStore.GetIcon("color")
		require.NoError(t, err, "Should be able to retrieve stored icon")
		assert.Equal(t, maxValidPNG, storedIconData, "Stored icon data should match uploaded data")
	})
}

func TestGetIcon(t *testing.T) {
	th := setupTestHelper(t)

	// Store a custom icon first
	customIconData := createTestPNGData(192, 192)
	err := th.p.pluginStore.StoreIcon("color", customIconData)
	require.NoError(t, err)

	t.Run("returns custom icon when available", func(t *testing.T) {
		// Create admin user for this test since getIcon requires admin permissions
		team := th.SetupTeam(t)
		admin := th.SetupUser(t, team)
		_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
		require.Nil(t, appErr)

		req := httptest.NewRequest(http.MethodGet, "/icons/color", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)
		w := httptest.NewRecorder()

		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get("Cache-Control"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, customIconData, body)
	})

	t.Run("returns default color icon when custom not available", func(t *testing.T) {
		// Create admin user for this test since getIcon requires admin permissions
		team := th.SetupTeam(t)
		admin := th.SetupUser(t, team)
		_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
		require.Nil(t, appErr)

		req := httptest.NewRequest(http.MethodGet, "/icons/outline", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)
		w := httptest.NewRecorder()

		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache, no-store, must-revalidate", resp.Header.Get("Cache-Control"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.True(t, len(body) > 0, "Should return default icon data")
	})

	t.Run("bad request for invalid icon type", func(t *testing.T) {
		// Create admin user for this test since getIcon requires admin permissions
		team := th.SetupTeam(t)
		admin := th.SetupUser(t, team)
		_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
		require.Nil(t, appErr)

		req := httptest.NewRequest(http.MethodGet, "/icons/invalid", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)
		w := httptest.NewRecorder()

		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Invalid icon type")
	})
}

func TestDeleteIcon(t *testing.T) {
	th := setupTestHelper(t)

	// Create admin user
	team := th.SetupTeam(t)
	admin := th.SetupUser(t, team)

	// Grant admin permissions
	_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
	require.Nil(t, appErr)

	// Create regular user
	regularUser := th.SetupUser(t, team)

	// Store a custom icon first
	customIconData := createTestPNGData(192, 192)
	err := th.p.pluginStore.StoreIcon("color", customIconData)
	require.NoError(t, err)

	t.Run("successful deletion of existing icon", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/icons/color", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"success":true`)
		assert.Contains(t, string(body), "Icon deleted successfully")

		// Verify icon was actually deleted
		_, err = th.p.pluginStore.GetIcon("color")
		assert.Error(t, err)
	})

	t.Run("successful deletion of non-existent icon", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/icons/outline", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"success":true`)
	})

	t.Run("unauthorized when no user ID provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/icons/color", nil)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("forbidden when user is not admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/icons/color", nil)
		req.Header.Set("Mattermost-User-ID", regularUser.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("bad request for invalid icon type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/icons/invalid", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Invalid icon type")
	})
}

func TestIconExists(t *testing.T) {
	th := setupTestHelper(t)

	// Create admin user
	team := th.SetupTeam(t)
	admin := th.SetupUser(t, team)

	// Grant admin permissions
	_, appErr := th.p.API.UpdateUserRoles(admin.Id, model.SystemAdminRoleId)
	require.Nil(t, appErr)

	// Create regular user
	regularUser := th.SetupUser(t, team)

	// Store a custom icon first
	customIconData := createTestPNGData(192, 192)
	err := th.p.pluginStore.StoreIcon("color", customIconData)
	require.NoError(t, err)

	t.Run("returns true when custom icon exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icons/color/exists", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"exists":true`)
	})

	t.Run("returns false when custom icon does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icons/outline/exists", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"exists":false`)
	})

	t.Run("unauthorized when no user ID provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icons/color/exists", nil)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("forbidden when user is not admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icons/color/exists", nil)
		req.Header.Set("Mattermost-User-ID", regularUser.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("bad request for invalid icon type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icons/invalid/exists", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Invalid icon type")
	})

	t.Run("returns false after icon is deleted", func(t *testing.T) {
		// First verify the icon exists
		req := httptest.NewRequest(http.MethodGet, "/icons/color/exists", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w := httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"exists":true`)

		// Delete the icon
		err = th.p.pluginStore.DeleteIcon("color")
		require.NoError(t, err)

		// Check that it no longer exists
		req = httptest.NewRequest(http.MethodGet, "/icons/color/exists", nil)
		req.Header.Set("Mattermost-User-ID", admin.Id)

		w = httptest.NewRecorder()
		th.p.apiHandler.ServeHTTP(w, req)

		resp = w.Result()
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"exists":false`)
	})
}
