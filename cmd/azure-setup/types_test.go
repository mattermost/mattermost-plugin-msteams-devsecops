// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequiredPermissions(t *testing.T) {
	permissions := getRequiredPermissions()

	// Should have exactly 3 permissions
	assert.Len(t, permissions, 3, "should have 3 required permissions")

	// Create a map for easier lookup
	permMap := make(map[string]requiredPermission)
	for _, perm := range permissions {
		permMap[perm.Name] = perm
	}

	// Verify User.Read permission
	userRead, ok := permMap["User.Read"]
	assert.True(t, ok, "should have User.Read permission")
	assert.Equal(t, GraphResourceID, userRead.ResourceAppID)
	assert.Equal(t, PermissionUserReadID, userRead.ResourceID)
	assert.Equal(t, PermissionTypeScope, userRead.Type)

	// Verify TeamsActivity.Send permission
	teamsActivity, ok := permMap["TeamsActivity.Send"]
	assert.True(t, ok, "should have TeamsActivity.Send permission")
	assert.Equal(t, GraphResourceID, teamsActivity.ResourceAppID)
	assert.Equal(t, PermissionTeamsActivitySendID, teamsActivity.ResourceID)
	assert.Equal(t, PermissionTypeRole, teamsActivity.Type)

	// Verify AppCatalog.Read.All permission
	appCatalog, ok := permMap["AppCatalog.Read.All"]
	assert.True(t, ok, "should have AppCatalog.Read.All permission")
	assert.Equal(t, GraphResourceID, appCatalog.ResourceAppID)
	assert.Equal(t, PermissionAppCatalogReadAllID, appCatalog.ResourceID)
	assert.Equal(t, PermissionTypeRole, appCatalog.Type)
}

func TestGetPreAuthorizedClients(t *testing.T) {
	clients := getPreAuthorizedClients()

	// Should have exactly 4 pre-authorized clients
	assert.Len(t, clients, 4, "should have 4 pre-authorized clients")

	// Verify all expected client IDs are present
	expectedClients := []string{
		ClientIDTeamsWeb,
		ClientIDTeamsDesktop,
		ClientIDOutlookWeb,
		ClientIDOutlookDesktop,
	}

	for _, expected := range expectedClients {
		assert.Contains(t, clients, expected, "should contain client ID: %s", expected)
	}
}

func TestPermissionConstants(t *testing.T) {
	// Verify Graph Resource ID is correct
	assert.Equal(t, "00000003-0000-0000-c000-000000000000", GraphResourceID)

	// Verify permission IDs are valid UUIDs (basic format check)
	uuidPattern := "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"

	assert.Regexp(t, uuidPattern, PermissionUserReadID, "User.Read ID should be a valid UUID")
	assert.Regexp(t, uuidPattern, PermissionTeamsActivitySendID, "TeamsActivity.Send ID should be a valid UUID")
	assert.Regexp(t, uuidPattern, PermissionAppCatalogReadAllID, "AppCatalog.Read.All ID should be a valid UUID")

	// Verify permission types
	assert.Equal(t, "Scope", PermissionTypeScope)
	assert.Equal(t, "Role", PermissionTypeRole)
}

func TestPreAuthorizedClientConstants(t *testing.T) {
	// Verify all client IDs are valid UUIDs (basic format check)
	uuidPattern := "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"

	assert.Regexp(t, uuidPattern, ClientIDTeamsWeb, "Teams Web client ID should be a valid UUID")
	assert.Regexp(t, uuidPattern, ClientIDTeamsDesktop, "Teams Desktop client ID should be a valid UUID")
	assert.Regexp(t, uuidPattern, ClientIDOutlookWeb, "Outlook Web client ID should be a valid UUID")
	assert.Regexp(t, uuidPattern, ClientIDOutlookDesktop, "Outlook Desktop client ID should be a valid UUID")

	// Verify the specific known values
	assert.Equal(t, "5e3ce6c0-2b1f-4285-8d4b-75ee78787346", ClientIDTeamsWeb)
	assert.Equal(t, "1fec8e78-bce4-4aaf-ab1b-5451cc387264", ClientIDTeamsDesktop)
	assert.Equal(t, "bc59ab01-8403-45c6-8796-ac3ef710b3e3", ClientIDOutlookWeb)
	assert.Equal(t, "d3590ed6-52b3-4102-aeff-aad2292ab01c", ClientIDOutlookDesktop)
}

func TestScopeConstants(t *testing.T) {
	assert.Equal(t, "access_as_user", ScopeName)
	assert.Equal(t, "Allow the app to access Mattermost on behalf of the signed-in user", ScopeDescription)
	assert.Equal(t, "Admins and users", ScopeUserConsent)
}

func TestSetupConfigDefaults(t *testing.T) {
	config := &SetupConfig{
		MattermostSiteURL: "https://mattermost.example.com",
		AppName:           "Test App",
		SecretExpiration:  12,
		DryRun:            false,
		NonInteractive:    false,
		Verbose:           false,
		OutputFormat:      "human",
	}

	// Verify basic initialization
	assert.NotNil(t, config)
	assert.Equal(t, "https://mattermost.example.com", config.MattermostSiteURL)
	assert.Equal(t, "Test App", config.AppName)
	assert.Equal(t, 12, config.SecretExpiration)
	assert.False(t, config.DryRun)
	assert.False(t, config.NonInteractive)
	assert.False(t, config.Verbose)
	assert.Equal(t, "human", config.OutputFormat)

	// Verify rollback slice is initialized
	config.rollback = []func() error{}
	assert.NotNil(t, config.rollback)
	assert.Len(t, config.rollback, 0)
}

func TestSetupResultStructure(t *testing.T) {
	result := &SetupResult{
		Success:             true,
		Message:             "Test message",
		ApplicationClientID: "client-123",
		TenantID:            "tenant-123",
		ClientSecret:        "secret-123",
		SecretExpiration:    "2026-01-15",
		ApplicationID:       "app-123",
		ApplicationName:     "Test App",
		ApplicationIDURI:    "api://test.com/client-123",
		Created:             true,
		DryRun:              false,
	}

	// Verify all fields are accessible
	assert.True(t, result.Success)
	assert.Equal(t, "Test message", result.Message)
	assert.Equal(t, "client-123", result.ApplicationClientID)
	assert.Equal(t, "tenant-123", result.TenantID)
	assert.Equal(t, "secret-123", result.ClientSecret)
	assert.Equal(t, "2026-01-15", result.SecretExpiration)
	assert.Equal(t, "app-123", result.ApplicationID)
	assert.Equal(t, "Test App", result.ApplicationName)
	assert.Equal(t, "api://test.com/client-123", result.ApplicationIDURI)
	assert.True(t, result.Created)
	assert.False(t, result.DryRun)
}
