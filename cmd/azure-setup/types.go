// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

// Microsoft Graph API Permission IDs and Types
const (
	// Graph Resource ID (Microsoft Graph)
	GraphResourceID = "00000003-0000-0000-c000-000000000000"

	// Required Permission IDs from Microsoft Graph
	PermissionUserReadID          = "e1fe6dd8-ba31-4d61-89e7-88639da4683d" // User.Read (Delegated)
	PermissionTeamsActivitySendID = "a267235f-af13-44dc-8385-c1dc93023186" // TeamsActivity.Send (Application)
	PermissionAppCatalogReadAllID = "e12dae10-5a57-4817-b79d-dfbec5348930" // AppCatalog.Read.All (Application)

	// Permission Types
	PermissionTypeScope = "Scope" // Delegated permission
	PermissionTypeRole  = "Role"  // Application permission
)

// Pre-authorized client application IDs for Microsoft Teams and Outlook
const (
	// Microsoft Teams
	ClientIDTeamsWeb     = "5e3ce6c0-2b1f-4285-8d4b-75ee78787346"
	ClientIDTeamsDesktop = "1fec8e78-bce4-4aaf-ab1b-5451cc387264"

	// Microsoft Outlook
	ClientIDOutlookWeb     = "bc59ab01-8403-45c6-8796-ac3ef710b3e3"
	ClientIDOutlookDesktop = "d3590ed6-52b3-4102-aeff-aad2292ab01c"
)

// Scope configuration
const (
	ScopeName        = "access_as_user"
	ScopeDescription = "Allow the app to access Mattermost on behalf of the signed-in user"
	ScopeUserConsent = "Admins and users"
)

// SetupConfig holds the configuration for the Azure setup process
type SetupConfig struct {
	// User inputs
	TenantID          string
	MattermostSiteURL string
	AppName           string
	ClientID          string // Optional: for updating existing app
	SecretExpiration  int    // Duration in months (default: 12)

	// Flags
	DryRun         bool
	NonInteractive bool
	Verbose        bool
	OutputFormat   string // "human", "json", "env"

	// Internal state
	ctx        context.Context
	credential azcore.TokenCredential
	createdApp *models.Application
	rollback   []func() error
}

// SetupResult contains the results of the Azure setup operation
type SetupResult struct {
	Success bool
	Message string

	// Azure App Registration Details
	ApplicationClientID string
	TenantID            string
	ClientSecret        string
	SecretExpiration    string

	// App Details
	ApplicationID    string
	ApplicationName  string
	ApplicationIDURI string

	// Operation Details
	Created bool // true if created new, false if updated existing
	DryRun  bool
}

// requiredPermission represents a Graph API permission that needs to be configured
type requiredPermission struct {
	ResourceAppID string // The app ID of the resource (e.g., Microsoft Graph)
	ResourceID    string // The permission ID
	Type          string // "Scope" or "Role"
	Name          string // Human-readable name for logging
}

// getRequiredPermissions returns the list of permissions needed for the plugin
func getRequiredPermissions() []requiredPermission {
	return []requiredPermission{
		{
			ResourceAppID: GraphResourceID,
			ResourceID:    PermissionUserReadID,
			Type:          PermissionTypeScope,
			Name:          "User.Read",
		},
		{
			ResourceAppID: GraphResourceID,
			ResourceID:    PermissionTeamsActivitySendID,
			Type:          PermissionTypeRole,
			Name:          "TeamsActivity.Send",
		},
		{
			ResourceAppID: GraphResourceID,
			ResourceID:    PermissionAppCatalogReadAllID,
			Type:          PermissionTypeRole,
			Name:          "AppCatalog.Read.All",
		},
	}
}

// getPreAuthorizedClients returns the list of Microsoft client IDs that should be pre-authorized
func getPreAuthorizedClients() []string {
	return []string{
		ClientIDTeamsWeb,
		ClientIDTeamsDesktop,
		ClientIDOutlookWeb,
		ClientIDOutlookDesktop,
	}
}
