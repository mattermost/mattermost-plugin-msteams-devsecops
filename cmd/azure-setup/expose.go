// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
)

// configureAPIExposure configures the Application ID URI, scopes, and pre-authorized applications
func configureAPIExposure(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *SetupConfig, app models.Applicationable) error {
	if config.Verbose {
		fmt.Println("🌐 Configuring API exposure...")
	}

	// Build the Application ID URI
	appIDURI, err := buildApplicationIDURI(config.MattermostSiteURL, *app.GetAppId())
	if err != nil {
		return errors.Wrap(err, "failed to build Application ID URI")
	}

	if config.DryRun {
		fmt.Println("   [DRY RUN] Would configure API exposure:")
		fmt.Printf("      Application ID URI: %s\n", appIDURI)
		fmt.Printf("      Scope: %s\n", ScopeName)
		fmt.Println("      Pre-authorized clients:")
		for _, clientID := range getPreAuthorizedClients() {
			fmt.Printf("         - %s\n", clientID)
		}
		return nil
	}

	// Create the api configuration
	api := models.NewApiApplication()

	// Set the Application ID URI (identifier)
	identifierUris := []string{appIDURI}
	appUpdate := models.NewApplication()
	appUpdate.SetIdentifierUris(identifierUris)

	// Create the access_as_user scope
	scopeID := uuid.New()
	scope := models.NewPermissionScope()
	scope.SetId(&scopeID)

	scopeName := ScopeName
	scope.SetValue(&scopeName)

	adminConsentDisplayName := "Access Mattermost"
	scope.SetAdminConsentDisplayName(&adminConsentDisplayName)

	adminConsentDescription := ScopeDescription
	scope.SetAdminConsentDescription(&adminConsentDescription)

	userConsentDisplayName := "Access Mattermost"
	scope.SetUserConsentDisplayName(&userConsentDisplayName)

	userConsentDescription := "Allow the app to access Mattermost on your behalf"
	scope.SetUserConsentDescription(&userConsentDescription)

	scopeType := "User"
	scope.SetTypeEscaped(&scopeType)

	isEnabled := true
	scope.SetIsEnabled(&isEnabled)

	oauth2PermissionScopes := []models.PermissionScopeable{scope}
	api.SetOauth2PermissionScopes(oauth2PermissionScopes)

	// Add pre-authorized applications
	preAuthorizedApps, err := buildPreAuthorizedApplications(scopeID)
	if err != nil {
		return err
	}
	api.SetPreAuthorizedApplications(preAuthorizedApps)

	appUpdate.SetApi(api)

	// Update the application
	_, err = client.Applications().ByApplicationId(*app.GetId()).Patch(ctx, appUpdate, nil)
	if err != nil {
		return errors.Wrap(err, "failed to configure API exposure")
	}

	if config.Verbose {
		fmt.Println("✅ API exposure configured:")
		fmt.Printf("   ✓ Application ID URI: %s\n", appIDURI)
		fmt.Printf("   ✓ Scope: %s\n", ScopeName)
		fmt.Println("   ✓ Pre-authorized clients:")
		fmt.Println("      - Microsoft Teams Web")
		fmt.Println("      - Microsoft Teams Desktop")
		fmt.Println("      - Microsoft Outlook Web")
		fmt.Println("      - Microsoft Outlook Desktop")
	}

	return nil
}

// buildPreAuthorizedApplications creates the list of pre-authorized applications
// Returns an error if any hardcoded client ID fails to parse (which should never happen)
func buildPreAuthorizedApplications(scopeID uuid.UUID) ([]models.PreAuthorizedApplicationable, error) {
	clientIDs := getPreAuthorizedClients()
	var preAuthorizedApps []models.PreAuthorizedApplicationable

	for _, clientIDStr := range clientIDs {
		clientID, err := uuid.Parse(clientIDStr)
		if err != nil {
			// This should never happen with hardcoded Microsoft client IDs
			// If it does, it indicates a bug in our constants
			return nil, errors.Wrapf(err, "BUG: invalid hardcoded pre-authorized client ID %s", clientIDStr)
		}

		preAuthApp := models.NewPreAuthorizedApplication()
		parsedClientIDStr := clientID.String()
		preAuthApp.SetAppId(&parsedClientIDStr)

		// Add the scope ID to the delegated permission IDs
		delegatedPermissionIDs := []string{scopeID.String()}
		preAuthApp.SetDelegatedPermissionIds(delegatedPermissionIDs)

		preAuthorizedApps = append(preAuthorizedApps, preAuthApp)
	}

	return preAuthorizedApps, nil
}
