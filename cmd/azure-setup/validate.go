// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
)

// validateInputs validates all user inputs before proceeding
func validateInputs(config *SetupConfig) error {
	// Validate Mattermost site URL
	if config.MattermostSiteURL == "" {
		return errors.New("Mattermost site URL is required")
	}

	u, err := url.Parse(config.MattermostSiteURL)
	if err != nil {
		return errors.Wrap(err, "invalid Mattermost site URL")
	}

	if u.Scheme == "" || u.Host == "" {
		return errors.New("Mattermost site URL must include protocol (https://) and hostname")
	}

	if u.Scheme != "https" {
		return errors.New("Mattermost site URL must use HTTPS protocol")
	}

	// Validate app name
	if config.AppName == "" {
		return errors.New("application name is required")
	}

	if len(config.AppName) < 3 {
		return errors.New("application name must be at least 3 characters")
	}

	// Validate secret expiration
	if config.SecretExpiration <= 0 {
		config.SecretExpiration = 12 // Default to 12 months
	}

	if config.SecretExpiration > 24 {
		return errors.New("secret expiration cannot exceed 24 months")
	}

	return nil
}

// validatePermissions checks if the authenticated user has the necessary permissions
func validatePermissions(ctx context.Context, client *msgraphsdk.GraphServiceClient, verbose bool) error {
	if verbose {
		fmt.Println("🔍 Checking user permissions...")
	}

	// Get the current user's service principal to check permissions
	// We need to verify the user can create/manage applications
	me, err := client.Me().Get(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get current user information")
	}

	if verbose {
		fmt.Printf("   Authenticated as: %s\n", *me.GetUserPrincipalName())
	}

	// Check if user has the necessary directory roles or permissions
	// For creating apps, user needs one of:
	// - Application Administrator role
	// - Cloud Application Administrator role
	// - Global Administrator role
	// OR
	// - Users can create applications setting enabled

	memberOf, err := client.Me().MemberOf().Get(ctx, nil)
	if err != nil {
		// Always show this warning as it's important for users to know
		fmt.Println("⚠️  Warning: Could not check directory roles")
		if verbose {
			fmt.Printf("   Error details: %v\n", err)
		}
		// Don't fail here - proceed and let the actual operations fail if needed
		return nil
	}

	// Check for admin roles
	hasAdminRole := false
	if memberOf != nil && memberOf.GetValue() != nil {
		for _, member := range memberOf.GetValue() {
			if directoryRole, ok := member.(models.DirectoryRoleable); ok {
				roleTemplate := directoryRole.GetRoleTemplateId()
				if roleTemplate != nil {
					roleID := *roleTemplate
					// Check for known admin role template IDs
					if isApplicationAdminRole(roleID) {
						hasAdminRole = true
						if verbose {
							fmt.Printf("   ✅ User has admin role: %s\n", *directoryRole.GetDisplayName())
						}
						break
					}
				}
			}
		}
	}

	if !hasAdminRole {
		// Always show this warning as it's critical for users to know
		fmt.Println("⚠️  Warning: User may not have Application Administrator permissions")
		fmt.Println("   Setup will proceed, but may fail if permissions are insufficient")
		fmt.Println("   Required role: Application Administrator, Cloud Application Administrator, or Global Administrator")
	}

	if verbose && hasAdminRole {
		fmt.Println("✅ User has sufficient permissions")
	}

	return nil
}

// isApplicationAdminRole checks if a role template ID corresponds to an application admin role
func isApplicationAdminRole(roleTemplateID string) bool {
	// Known Azure AD role template IDs for application management
	adminRoles := map[string]bool{
		"9b895d92-2cd3-44c7-9d02-a6ac2d5ea5c3": true, // Application Administrator
		"158c047a-c907-4556-b7ef-446551a6b5f7": true, // Cloud Application Administrator
		"62e90394-69f5-4237-9190-012177145e10": true, // Global Administrator
	}
	return adminRoles[roleTemplateID]
}

// checkExistingApp checks if an application with the given name or client ID already exists
func checkExistingApp(ctx context.Context, client *msgraphsdk.GraphServiceClient, appName, clientID string, verbose bool) (models.Applicationable, error) {
	if verbose {
		fmt.Println("🔍 Checking for existing application...")
	}

	var filter string
	if clientID != "" {
		// Search by client ID (appId field)
		// Escape single quotes for OData filter
		escapedClientID := strings.ReplaceAll(clientID, "'", "''")
		filter = fmt.Sprintf("appId eq '%s'", escapedClientID)
	} else {
		// Search by display name
		// Escape single quotes for OData filter
		escapedAppName := strings.ReplaceAll(appName, "'", "''")
		filter = fmt.Sprintf("displayName eq '%s'", escapedAppName)
	}

	apps, err := client.Applications().Get(ctx, &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &applications.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to search for existing application")
	}

	if apps == nil || apps.GetValue() == nil || len(apps.GetValue()) == 0 {
		if verbose {
			fmt.Println("   No existing application found")
		}
		return nil, nil
	}

	existingApp := apps.GetValue()[0]
	if verbose {
		fmt.Printf("   ✅ Found existing application: %s (ID: %s)\n",
			*existingApp.GetDisplayName(),
			*existingApp.GetAppId())
	}

	return existingApp, nil
}

// extractHostnameAndPath extracts the hostname and path from a Mattermost URL
// Returns hostname (without protocol) and path
func extractHostnameAndPath(siteURL string) (string, string, error) {
	u, err := url.Parse(siteURL)
	if err != nil {
		return "", "", err
	}

	hostname := u.Host
	path := strings.TrimSuffix(u.Path, "/")

	return hostname, path, nil
}

// buildApplicationIDURI builds the Application ID URI in the format: api://hostname/path/clientID
func buildApplicationIDURI(siteURL, clientID string) (string, error) {
	hostname, path, err := extractHostnameAndPath(siteURL)
	if err != nil {
		return "", err
	}

	// Build the URI
	if path == "" || path == "/" {
		return fmt.Sprintf("api://%s/%s", hostname, clientID), nil
	}

	return fmt.Sprintf("api://%s%s/%s", hostname, path, clientID), nil
}

// getTenantID retrieves the tenant ID from the Azure organization
func getTenantID(ctx context.Context, client *msgraphsdk.GraphServiceClient, verbose bool) (string, error) {
	if verbose {
		fmt.Println("🔍 Retrieving tenant ID from Azure...")
	}

	// Get organization details to retrieve tenant ID
	org, err := client.Organization().Get(ctx, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to get organization information")
	}

	if org == nil || org.GetValue() == nil || len(org.GetValue()) == 0 {
		return "", errors.New("no organization information found")
	}

	orgInfo := org.GetValue()[0]
	tenantID := orgInfo.GetId()
	if tenantID == nil || *tenantID == "" {
		return "", errors.New("organization ID is empty")
	}

	if verbose {
		fmt.Printf("   ✅ Tenant ID: %s\n", *tenantID)
	}

	return *tenantID, nil
}
