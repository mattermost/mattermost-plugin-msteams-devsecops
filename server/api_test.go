// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/store/pluginstore"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestIFrame(t *testing.T) {
	th := setupTestHelper(t)

	t.Run("returns iframe HTML", func(t *testing.T) {
		// Setup
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/mattermostTab", nil)

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

		// Check for CSP headers
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "style-src 'nonce-")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "script-src https://res.cdn.office.net 'nonce-")
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))

		// Check for MMEMBED cookie
		cookies := resp.Cookies()
		var embedCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "MMEMBED" {
				embedCookie = cookie
				break
			}
		}
		require.NotNil(t, embedCookie)
		assert.Equal(t, "1", embedCookie.Value)
		assert.Equal(t, "/", embedCookie.Path)
		assert.True(t, embedCookie.Secure)
		assert.Equal(t, http.SameSiteNoneMode, embedCookie.SameSite)

		// Check response body contains expected HTML
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "<html")
		assert.Contains(t, string(body), "</html>")
	})
}

func TestAuthenticate(t *testing.T) {
	th := setupTestHelper(t)

	t.Run("redirects when user is already logged in", func(t *testing.T) {
		// Setup
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/authenticate", nil)

		// Create a user
		team := th.SetupTeam(t)
		user := th.SetupUser(t, team)

		// Store user in the plugin store
		err := th.p.pluginStore.StoreUser(pluginstore.NewUser(user.Id, "test-oid", user.Email))
		require.NoError(t, err)

		// Set the Mattermost-User-ID header to simulate a logged-in user
		r.Header.Set("Mattermost-User-ID", user.Id)

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		assert.Equal(t, "/", resp.Header.Get("Location"))
	})

	t.Run("returns error when token is missing", func(t *testing.T) {
		// Setup
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/authenticate", nil)

		// No Mattermost-User-ID header to simulate a non-logged-in user
		// No token in query params

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Missing token")
	})
}

func TestIframeNotificationPreview(t *testing.T) {
	t.Run("returns error when user is not authenticated", func(t *testing.T) {
		th := setupTestHelper(t)
		// Setup
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/notification_preview", nil)

		// No Mattermost-User-ID header to simulate a non-logged-in user

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "user not authenticated")
	})

	t.Run("returns error when post_id is missing", func(t *testing.T) {
		th := setupTestHelper(t)
		// Setup
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/notification_preview", nil)

		// Create a user
		team := th.SetupTeam(t)
		user := th.SetupUser(t, team)

		// Set the Mattermost-User-ID header to simulate a logged-in user
		r.Header.Set("Mattermost-User-ID", user.Id)

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "post_id is required")
	})

	t.Run("returns error when user does not have read access to post channel", func(t *testing.T) {
		th := setupTestHelper(t)
		// Setup
		team := th.SetupTeam(t)
		user := th.SetupUser(t, team)

		// Create a channel
		channel, appErr := th.p.API.CreateChannel(&model.Channel{
			TeamId:      team.Id,
			Type:        model.ChannelTypeOpen,
			DisplayName: "Test Channel",
			Name:        "test-channel",
			Header:      "Test Header",
			Purpose:     "Test Purpose",
		})
		require.Nil(t, appErr)

		// Create a post
		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    user.Id,
			ChannelId: channel.Id,
			Message:   "Test message for notification preview",
		})
		require.Nil(t, appErr)

		// Setup request with post_id
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/notification_preview?post_id="+post.Id, nil)
		r.Header.Set("Mattermost-User-ID", user.Id)

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "user does not have access to the post")
	})

	t.Run("returns HTML preview when post exists", func(t *testing.T) {
		th := setupTestHelper(t)
		// Setup
		team := th.SetupTeam(t)
		user := th.SetupUser(t, team)
		user2 := th.SetupUser(t, team)

		// Create a channel
		channel, appErr := th.p.API.CreateChannel(&model.Channel{
			TeamId:      team.Id,
			Type:        model.ChannelTypeOpen,
			DisplayName: "Test Channel",
			Name:        "test-channel",
			Header:      "Test Header",
			Purpose:     "Test Purpose",
		})
		require.Nil(t, appErr)

		// add user to channel
		_, appErr = th.p.API.AddUserToChannel(channel.Id, user.Id, user2.Id)
		require.Nil(t, appErr)

		// Create a post
		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    user.Id,
			ChannelId: channel.Id,
			Message:   "Test message for notification preview",
		})
		require.Nil(t, appErr)

		// Setup request with post_id
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/iframe/notification_preview?post_id="+post.Id, nil)
		r.Header.Set("Mattermost-User-ID", user.Id)

		// Execute
		th.p.apiHandler.ServeHTTP(w, r)

		// Assert
		resp := w.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

		// Check for CSP headers
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "style-src 'nonce-")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "script-src https://res.cdn.office.net https://cdn.jsdelivr.net 'nonce-")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "script-src-attr 'nonce-")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "connect-src https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "img-src 'self'")
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "report-to csp-endpoint")
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))

		// Check for Report-To header
		require.Contains(t, resp.Header.Get("Report-To"), `{"group":"csp-endpoint","max_age":10886400,"endpoints":[{"url":"/plugins/`+manifest.Id+`/csp-report"}]}`)

		// Check response body contains expected HTML
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "<html")
		assert.Contains(t, string(body), post.Message)
	})
}
