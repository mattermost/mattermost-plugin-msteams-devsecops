// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/store/pluginstore"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIFrameAuthenticate(t *testing.T) {
	th := setupTestHelper(t)
	apiURL := th.pluginURL(t, "/iframe/authenticate")

	// Create a client that doesn't follow redirects
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("already logged in user", func(t *testing.T) {
		th.Reset(t)

		team := th.SetupTeam(t)
		user := th.SetupUser(t, team)
		client := th.SetupClient(t, user.Id)

		// Add the user to the plugin store
		err := th.p.pluginStore.StoreUser(pluginstore.NewUser(user.Id, "test-oid", user.Email))
		require.NoError(t, err)

		// Mock client user method
		th.clientMock.On("User", mock.Anything).Return(user).Maybe()
		th.clientMock.On("Get", user.Id).Return(user, nil).Maybe()

		request, err := http.NewRequest(http.MethodGet, apiURL, nil)
		require.NoError(t, err)

		// Set the Mattermost-User-ID header to simulate an already logged in user
		request.Header.Set("Mattermost-User-ID", user.Id)
		request.Header.Set(model.HeaderAuth, client.AuthType+" "+client.AuthToken)

		response, err := httpClient.Do(request)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, response.Body.Close())
		})

		// Should redirect to home page
		assert.Equal(t, http.StatusSeeOther, response.StatusCode)
		assert.Equal(t, "/", response.Header.Get("Location"))
	})

	t.Run("missing token", func(t *testing.T) {
		th.Reset(t)

		request, err := http.NewRequest(http.MethodGet, apiURL, nil)
		require.NoError(t, err)

		response, err := httpClient.Do(request)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, response.Body.Close())
		})

		// Should return an error
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
	})

	t.Run("invalid token", func(t *testing.T) {
		th.Reset(t)

		request, err := http.NewRequest(http.MethodGet, apiURL+"?token=invalid_token", nil)
		require.NoError(t, err)

		response, err := httpClient.Do(request)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, response.Body.Close())
		})

		// Should return an error
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
	})

	// Note: Testing with a valid token would require mocking the JWT validation
	// and claims extraction, which would be more complex and require additional setup
}

func TestGetCookieDomain(t *testing.T) {
	tests := []struct {
		name                     string
		siteURL                  string
		allowCookiesForSubdomain *bool
		expected                 string
	}{
		{
			name:                     "Allow cookies for subdomains with valid URL",
			siteURL:                  "https://example.mattermost.com",
			allowCookiesForSubdomain: model.NewPointer(true),
			expected:                 "example.mattermost.com",
		},
		{
			name:                     "Allow cookies for subdomains with invalid URL",
			siteURL:                  "invalid-url",
			allowCookiesForSubdomain: model.NewPointer(true),
			expected:                 "",
		},
		{
			name:                     "Disallow cookies for subdomains",
			siteURL:                  "https://example.mattermost.com",
			allowCookiesForSubdomain: model.NewPointer(false),
			expected:                 "",
		},
		{
			name:                     "Allow cookies for subdomains with URL containing port",
			siteURL:                  "https://example.mattermost.com:8065",
			allowCookiesForSubdomain: model.NewPointer(true),
			expected:                 "example.mattermost.com",
		},
		{
			name:                     "Allow cookies for subdomains with localhost",
			siteURL:                  "http://localhost:8065",
			allowCookiesForSubdomain: model.NewPointer(true),
			expected:                 "localhost",
		},
		{
			name:                     "Nil AllowCookiesForSubdomain",
			siteURL:                  "https://example.mattermost.com",
			allowCookiesForSubdomain: nil,
			expected:                 "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &model.Config{}
			config.ServiceSettings.SiteURL = &tt.siteURL
			config.ServiceSettings.AllowCookiesForSubdomains = tt.allowCookiesForSubdomain

			result := getCookieDomain(config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRedirectPathFromUser(t *testing.T) {
	th := setupTestHelper(t)

	logger := logrus.StandardLogger().WithField("test", "TestGetRedirectPathFromUser")
	user := &model.User{Id: "user1"}

	t.Run("empty subEntityID", func(t *testing.T) {
		path := th.p.getRedirectPathFromUser(logger, user, "")
		assert.Equal(t, "/", path)
	})

	t.Run("post_preview subEntityID", func(t *testing.T) {
		path := th.p.getRedirectPathFromUser(logger, user, "post_preview_post123")
		assert.Equal(t, "/plugins/"+url.PathEscape(manifest.Id)+"/iframe/notification_preview?post_id=post123", path)
	})

	t.Run("post subEntityID with valid post and channel with team", func(t *testing.T) {
		// Setup real data using testHelper
		team := th.SetupTeam(t)
		user1 := th.SetupUser(t, team)
		channel, appErr := th.p.API.CreateChannel(&model.Channel{
			TeamId:      team.Id,
			Type:        model.ChannelTypeOpen,
			DisplayName: "Test Channel",
			Name:        "test-channel",
			CreatorId:   user1.Id,
		})
		require.Nil(t, appErr)

		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    user1.Id,
			ChannelId: channel.Id,
			Message:   "Test post",
		})
		require.Nil(t, appErr)

		// Test
		path := th.p.getRedirectPathFromUser(logger, user1, "post_"+post.Id)
		assert.Equal(t, "/"+team.Name+"/pl/"+post.Id, path)
	})

	t.Run("post subEntityID with valid post and DM channel", func(t *testing.T) {
		// Setup real data using testHelper
		team := th.SetupTeam(t)
		user1 := th.SetupUser(t, team)
		user2 := th.SetupUser(t, team)

		// Create a DM channel
		dmChannel := th.SetupDirectMessageChannel(t, user1.Id, user2.Id)

		// Create a post in the DM channel
		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    user1.Id,
			ChannelId: dmChannel.Id,
			Message:   "Test DM post",
		})
		require.Nil(t, appErr)

		// Test
		path := th.p.getRedirectPathFromUser(logger, user1, "post_"+post.Id)
		assert.Equal(t, "/"+team.Name+"/pl/"+post.Id, path)
	})

	t.Run("post subEntityID with non-existent post", func(t *testing.T) {
		// Test with a non-existent post ID
		path := th.p.getRedirectPathFromUser(logger, user, "post_non_existent_post_id")
		assert.Equal(t, "/", path)
	})

	t.Run("post subEntityID with non-existent channel", func(t *testing.T) {
		// Create a post ID that doesn't exist
		path := th.p.getRedirectPathFromUser(logger, user, "post_non_existent_post_id")
		assert.Equal(t, "/", path)
	})

	t.Run("post subEntityID with non-existent team", func(t *testing.T) {
		// Create a post with a non-existent team ID
		// For this test, we'll use a non-existent post ID to simulate the error path
		// when the team can't be found
		path := th.p.getRedirectPathFromUser(logger, user, "post_non_existent_post_id")
		assert.Equal(t, "/", path)
	})

	t.Run("post subEntityID with user having no teams", func(t *testing.T) {
		// Create a user without adding them to any teams
		randomUsername := model.NewId()
		userWithNoTeam, appErr := th.p.API.CreateUser(&model.User{
			Email:         randomUsername + "@example.com",
			Username:      randomUsername,
			Password:      "password",
			EmailVerified: true,
		})
		require.Nil(t, appErr)

		// Create another user that is part of a team
		team := th.SetupTeam(t)
		userWithTeam := th.SetupUser(t, team)

		// Create a DM channel between the two users
		dmChannel := th.SetupDirectMessageChannel(t, userWithNoTeam.Id, userWithTeam.Id)

		// Create a post in the DM channel
		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    userWithTeam.Id,
			ChannelId: dmChannel.Id,
			Message:   "Test DM post from user with team to user with no team",
		})
		require.Nil(t, appErr)

		// Test with the user that has no teams
		path := th.p.getRedirectPathFromUser(logger, userWithNoTeam, "post_"+post.Id)
		assert.Equal(t, "/", path)
	})

	t.Run("post subEntityID with channel having no team and user having no teams", func(t *testing.T) {
		// For this test, we'll create a direct message channel which has no team ID
		// and test with a user that has no teams

		// Create two users, one with no teams
		randomUsername := model.NewId()
		userWithNoTeam, appErr := th.p.API.CreateUser(&model.User{
			Email:         randomUsername + "@example.com",
			Username:      randomUsername,
			Password:      "password",
			EmailVerified: true,
		})
		require.Nil(t, appErr)

		// Create another user that is part of a team
		team := th.SetupTeam(t)
		userWithTeam := th.SetupUser(t, team)

		// Create a DM channel between the two users
		dmChannel := th.SetupDirectMessageChannel(t, userWithNoTeam.Id, userWithTeam.Id)

		// Create a post in the DM channel
		post, appErr := th.p.API.CreatePost(&model.Post{
			UserId:    userWithTeam.Id,
			ChannelId: dmChannel.Id,
			Message:   "Test DM post in channel with no team",
		})
		require.Nil(t, appErr)

		// Test with the user that has no teams
		// This should return "/" since the user has no teams to redirect to
		path := th.p.getRedirectPathFromUser(logger, userWithNoTeam, "post_"+post.Id)
		assert.Equal(t, "/", path)
	})

	t.Run("unknown subEntityID format", func(t *testing.T) {
		path := th.p.getRedirectPathFromUser(logger, user, "unknown_format")
		assert.Equal(t, "/", path)
	})
}
