// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"github.com/pkg/errors"
)

// configureAPIPermissions adds the required API permissions to the application
func configureAPIPermissions(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *SetupConfig, app models.Applicationable) error {
	if config.Verbose {
		fmt.Println("🔑 Configuring API permissions...")
	}

	if config.DryRun {
		fmt.Println("   [DRY RUN] Would configure the following API permissions:")
		for _, perm := range getRequiredPermissions() {
			permType := "Delegated"
			if perm.Type == PermissionTypeRole {
				permType = "Application"
			}
			fmt.Printf("      - %s (%s)\n", perm.Name, permType)
		}
		return nil
	}

	// Build the required resource access list
	requiredResourceAccess, err := buildRequiredResourceAccess()
	if err != nil {
		return err
	}

	// Update the application with the required permissions
	appUpdate := models.NewApplication()
	appUpdate.SetRequiredResourceAccess(requiredResourceAccess)

	_, err = client.Applications().ByApplicationId(*app.GetId()).Patch(ctx, appUpdate, nil)
	if err != nil {
		return errors.Wrap(err, "failed to configure API permissions")
	}

	if config.Verbose {
		fmt.Println("✅ API permissions configured:")
		for _, perm := range getRequiredPermissions() {
			permType := "Delegated"
			if perm.Type == PermissionTypeRole {
				permType = "Application"
			}
			fmt.Printf("   ✓ %s (%s)\n", perm.Name, permType)
		}
	}

	// Create service principal to enable admin consent
	if err := ensureServicePrincipalExists(ctx, client, config, app); err != nil {
		if config.Verbose {
			fmt.Printf("   ⚠️  Warning: Could not create service principal: %v\n", err)
			fmt.Println("      Service principal may need to be created manually")
		}
	}

	return nil
}

// buildRequiredResourceAccess builds the required resource access list for Microsoft Graph
func buildRequiredResourceAccess() ([]models.RequiredResourceAccessable, error) {
	permissions := getRequiredPermissions()

	// Group permissions by resource app ID
	resourceMap := make(map[string][]models.ResourceAccessable)

	for _, perm := range permissions {
		permUUID, err := uuid.Parse(perm.ResourceID)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid permission ID: %s", perm.ResourceID)
		}

		resourceAccess := models.NewResourceAccess()
		resourceAccess.SetId(&permUUID)
		resourceAccess.SetTypeEscaped(&perm.Type)

		resourceMap[perm.ResourceAppID] = append(resourceMap[perm.ResourceAppID], resourceAccess)
	}

	// Build the RequiredResourceAccess list
	var requiredResourceAccess []models.RequiredResourceAccessable

	for resourceAppID, accesses := range resourceMap {
		resourceUUID, err := uuid.Parse(resourceAppID)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid resource app ID: %s", resourceAppID)
		}

		required := models.NewRequiredResourceAccess()
		resourceAppIDStr := resourceUUID.String()
		required.SetResourceAppId(&resourceAppIDStr)
		required.SetResourceAccess(accesses)

		requiredResourceAccess = append(requiredResourceAccess, required)
	}

	return requiredResourceAccess, nil
}

// ensureServicePrincipalExists creates a service principal for the application if it doesn't exist
// The service principal is required for admin consent to be granted
// Note: This does NOT automatically grant admin consent - that must be done manually in the Azure Portal
func ensureServicePrincipalExists(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *SetupConfig, app models.Applicationable) error {
	if config.Verbose {
		fmt.Println("🔐 Ensuring service principal exists for admin consent...")
	}

	appID := *app.GetAppId()

	// Validate UUID before using in filter
	if _, err := uuid.Parse(appID); err != nil {
		return errors.Wrap(err, "invalid application client ID format")
	}

	filter := fmt.Sprintf("appId eq '%s'", appID)

	servicePrincipals, err := client.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to check for existing service principal")
	}

	var spObjectID string
	if servicePrincipals == nil || servicePrincipals.GetValue() == nil || len(servicePrincipals.GetValue()) == 0 {
		// Create service principal
		if config.Verbose {
			fmt.Println("   Creating service principal...")
		}

		newSP := models.NewServicePrincipal()
		newSP.SetAppId(&appID)

		createdSP, err := client.ServicePrincipals().Post(ctx, newSP, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create service principal")
		}

		spObjectID = *createdSP.GetId()

		// Add service principal cleanup to rollback
		config.rollback = append(config.rollback, func() error {
			return deleteServicePrincipal(ctx, client, spObjectID, config.Verbose)
		})

		if config.Verbose {
			fmt.Println("   ✅ Service principal created")
		}
	} else {
		if config.Verbose {
			fmt.Println("   ✅ Service principal already exists")
		}
	}

	if config.Verbose {
		// Validate UUID before constructing URL
		if _, err := uuid.Parse(appID); err == nil {
			fmt.Printf("✅ Admin consent must be granted manually\n")
			fmt.Printf("   Visit: https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/~/CallAnAPI/appId/%s\n", appID)
		}
	}

	return nil
}

// deleteServicePrincipal deletes a service principal (used for rollback)
func deleteServicePrincipal(ctx context.Context, client *msgraphsdk.GraphServiceClient, objectID string, verbose bool) error {
	if verbose {
		fmt.Printf("🗑️  Rolling back: Deleting service principal %s\n", objectID)
	}

	err := client.ServicePrincipals().ByServicePrincipalId(objectID).Delete(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete service principal during rollback")
	}

	return nil
}
