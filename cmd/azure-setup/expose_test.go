// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigureAPIExposure_DryRun tests dry-run mode for API exposure configuration
func TestConfigureAPIExposure_DryRun(t *testing.T) {
	ctx := context.Background()
	config := &SetupConfig{
		MattermostSiteURL: "https://mattermost.example.com",
		AppName:           "Test App",
		Verbose:           false,
		DryRun:            true,
		rollback:          []func() error{},
	}

	app := models.NewApplication()
	displayName := "Test App"
	app.SetDisplayName(&displayName)
	appID := "abc-123-def-456"
	app.SetAppId(&appID)

	err := configureAPIExposure(ctx, nil, config, app)
	require.NoError(t, err, "Dry run should not return error")
}

// TestBuildPreAuthorizedApplications tests building pre-authorized apps list
func TestBuildPreAuthorizedApplications(t *testing.T) {
	t.Run("creates_correct_number_of_apps", func(t *testing.T) {
		scopeID := uuid.New()

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)

		// Should have 9 pre-authorized apps (Teams, Outlook, Office, Copilot)
		assert.Len(t, apps, 9)
	})

	t.Run("each_app_has_scope_id", func(t *testing.T) {
		scopeID := uuid.New()

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)

		for i, app := range apps {
			delegatedPerms := app.GetDelegatedPermissionIds()
			assert.Len(t, delegatedPerms, 1, "App %d should have exactly 1 delegated permission", i)
			assert.Equal(t, scopeID.String(), delegatedPerms[0], "App %d should have correct scope ID", i)
		}
	})

	t.Run("validates_hardcoded_client_ids", func(t *testing.T) {
		scopeID := uuid.New()

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)

		// Verify all client IDs are valid UUIDs
		for i, app := range apps {
			appID := app.GetAppId()
			require.NotNil(t, appID, "App %d should have app ID", i)

			_, err := uuid.Parse(*appID)
			assert.NoError(t, err, "App %d client ID should be valid UUID: %s", i, *appID)
		}
	})

	t.Run("includes_all_required_microsoft_clients", func(t *testing.T) {
		scopeID := uuid.New()

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)

		// Convert to map for easier checking
		appIDMap := make(map[string]bool)
		for _, app := range apps {
			appIDMap[*app.GetAppId()] = true
		}

		// Verify all expected clients are present
		expectedClients := getPreAuthorizedClients()
		for _, clientID := range expectedClients {
			assert.True(t, appIDMap[clientID], "Should include client %s", clientID)
		}
	})
}

// TestConfigureAPIExposure_ApplicationIDURI tests Application ID URI generation
func TestConfigureAPIExposure_ApplicationIDURI(t *testing.T) {
	tests := []struct {
		name        string
		siteURL     string
		clientID    string
		expectedURI string
	}{
		{
			name:        "simple_domain",
			siteURL:     "https://mattermost.example.com",
			clientID:    "abc-123",
			expectedURI: "api://mattermost.example.com/abc-123",
		},
		{
			name:        "domain_with_port",
			siteURL:     "https://mattermost.example.com:8065",
			clientID:    "abc-123",
			expectedURI: "api://mattermost.example.com:8065/abc-123",
		},
		{
			name:        "domain_with_path",
			siteURL:     "https://example.com/mattermost",
			clientID:    "abc-123",
			expectedURI: "api://example.com/mattermost/abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := buildApplicationIDURI(tt.siteURL, tt.clientID)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURI, uri)
		})
	}
}

// TestConfigureAPIExposure_ScopeConfiguration tests scope setup
func TestConfigureAPIExposure_ScopeConfiguration(t *testing.T) {
	t.Run("scope_constants_are_defined", func(t *testing.T) {
		assert.Equal(t, "access_as_user", ScopeName)
		assert.NotEmpty(t, ScopeDescription)
		assert.Contains(t, ScopeDescription, "Mattermost")
	})
}

// TestConfigureAPIExposure_ErrorHandling tests error scenarios
func TestConfigureAPIExposure_ErrorHandling(t *testing.T) {
	t.Run("invalid_site_url", func(t *testing.T) {
		ctx := context.Background()
		config := &SetupConfig{
			MattermostSiteURL: "://invalid-url",
			AppName:           "Test App",
			Verbose:           false,
			DryRun:            true,
		}

		app := models.NewApplication()
		displayName := "Test App"
		app.SetDisplayName(&displayName)
		appID := "abc-123"
		app.SetAppId(&appID)

		err := configureAPIExposure(ctx, nil, config, app)
		// buildApplicationIDURI should fail with truly malformed URL
		assert.Error(t, err)
	})
}

// TestConfigureAPIExposure_VerboseOutput tests verbose mode
func TestConfigureAPIExposure_VerboseOutput(t *testing.T) {
	ctx := context.Background()
	config := &SetupConfig{
		MattermostSiteURL: "https://mattermost.example.com",
		AppName:           "Test App",
		Verbose:           true,
		DryRun:            true,
		rollback:          []func() error{},
	}

	app := models.NewApplication()
	displayName := "Test App"
	app.SetDisplayName(&displayName)
	appID := "abc-123-def-456"
	app.SetAppId(&appID)

	err := configureAPIExposure(ctx, nil, config, app)
	require.NoError(t, err, "Verbose mode should not affect success")
}

// TestBuildPreAuthorizedApplications_ErrorHandling tests error scenarios
func TestBuildPreAuthorizedApplications_ErrorHandling(t *testing.T) {
	t.Run("valid_uuid_scope", func(t *testing.T) {
		scopeID := uuid.New()

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)
		assert.NotEmpty(t, apps)
	})

	t.Run("zero_uuid_scope", func(t *testing.T) {
		scopeID := uuid.Nil

		apps, err := buildPreAuthorizedApplications(scopeID)
		require.NoError(t, err)

		// Should still work with zero UUID
		for _, app := range apps {
			delegatedPerms := app.GetDelegatedPermissionIds()
			assert.Equal(t, uuid.Nil.String(), delegatedPerms[0])
		}
	})
}
