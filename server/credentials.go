// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/msteams/clientmodels"
)

const (
	ResourceAccessTypeScope = "Scope"
	ResourceAccessTypeRole  = "Role"
)

type expectedPermission struct {
	Name           string
	ResourceAccess clientmodels.ResourceAccess
}

// getResourceAccessKey makes a map key for the resource access that simplifies checking
// for the resource access in question. (Technically, we could use the struct itself, but
// this insulates us from unexpected upstream changes.)
func getResourceAccessKey(resourceAccess clientmodels.ResourceAccess) string {
	return fmt.Sprintf("%s+%s", resourceAccess.ID, resourceAccess.Type)
}

// describeResourceAccessType annotates the resource access type with the user facing term
// shown in the Azure Tenant UI (Application vs. Delegated).
func describeResourceAccessType(resourceAccess clientmodels.ResourceAccess) string {
	switch resourceAccess.Type {
	case ResourceAccessTypeRole:
		return "Role (Application)"
	case ResourceAccessTypeScope:
		return "Scope (Delegated)"
	default:
		return resourceAccess.Type
	}
}

// getExpectedPermissions returns the set of expected permissions, keyed by the
// name the enduser would expect to see in the Azure tenant.
func getExpectedPermissions() []expectedPermission {
	return []expectedPermission{
		{
			Name: "https://graph.microsoft.com/User.Read",
			ResourceAccess: clientmodels.ResourceAccess{
				ID:   "e1fe6dd8-ba31-4d61-89e7-88639da4683d",
				Type: "Scope",
			},
		},
		{
			Name: "https://graph.microsoft.com/TeamsActivity.Send",
			ResourceAccess: clientmodels.ResourceAccess{
				ID:   "a267235f-af13-44dc-8385-c1dc93023186",
				Type: "Role",
			},
		},
		{
			Name: "https://graph.microsoft.com/AppCatalog.Read.All",
			ResourceAccess: clientmodels.ResourceAccess{
				ID:   "e12dae10-5a57-4817-b79d-dfbec5348930",
				Type: "Role",
			},
		},
	}
}

func (p *Plugin) checkCredentials() {
	if p.disableCheckCredentials {
		p.API.LogDebug("Skipping credentials check: check is disabled")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			p.API.LogError("Recovering from panic", "panic", r, "stack", string(debug.Stack()))
		}
	}()

	p.API.LogInfo("Running the check credentials job")

	client := p.GetClientForApp()
	if client == nil {
		p.API.LogWarn("MS Teams client not available, skipping credentials check")
		return
	}

	app, err := client.GetApp(p.getConfiguration().M365ClientID)
	if err != nil {
		p.API.LogWarn("Failed to get app credentials", "error", err.Error())
		return
	}

	credentials := app.Credentials

	// We sort by earliest end date to cover the unlikely event we encounter two credentials
	// with the same hint when reporting the single metric below.
	sort.SliceStable(credentials, func(i, j int) bool {
		return credentials[i].EndDateTime.Before(credentials[j].EndDateTime)
	})

	found := false
	for _, credential := range credentials {
		if strings.HasPrefix(p.getConfiguration().M365ClientSecret, credential.Hint) {
			p.API.LogInfo("Found matching credential", "credential_name", credential.Name, "credential_id", credential.ID, "credential_end_date_time", credential.EndDateTime)

			if found {
				// If we happen to get more than one with the same hint, we'll have reported the metric of the
				// earlier one by virtue of the sort above, and we'll have the extra metadata we need in the logs.
				p.API.LogWarn("Found more than one secret with same hint", "credential_id", credential.ID)
			}

			// Note that we keep going to log all the credentials found.
			found = true
		} else {
			p.API.LogInfo("Found other credential", "credential_name", credential.Name, "credential_id", credential.ID, "credential_end_date_time", credential.EndDateTime)
		}
	}

	if !found {
		p.API.LogWarn("Failed to find credential matching configuration")
	}

	missingPermissions, redundantResourceAccess := p.checkPermissions(app)
	for _, permission := range missingPermissions {
		p.API.LogWarn(
			"Application missing required API Permission",
			"permission", permission.Name,
			"resource_id", permission.ResourceAccess.ID,
			"type", describeResourceAccessType(permission.ResourceAccess),
			"application_id", p.getConfiguration().M365ClientID,
		)
	}

	for _, resourceAccess := range redundantResourceAccess {
		p.API.LogWarn(
			"Application has redundant API Permission",
			"resource_id", resourceAccess.ID,
			"type", describeResourceAccessType(resourceAccess),
			"application_id", p.getConfiguration().M365ClientID,
		)
	}
}

func (p *Plugin) checkPermissions(app *clientmodels.App) ([]expectedPermission, []clientmodels.ResourceAccess) {
	// Build a map and log what we find at the same time.
	actualRequiredResources := make(map[string]clientmodels.ResourceAccess)
	for _, requiredResource := range app.RequiredResources {
		actualRequiredResources[getResourceAccessKey(requiredResource)] = requiredResource
		p.API.LogDebug(
			"Found API Permission",
			"resource_id", requiredResource.ID,
			"type", describeResourceAccessType(requiredResource),
			"application_id", p.getConfiguration().M365ClientID,
		)
	}

	expectedPermissions := getExpectedPermissions()
	expectedPermissionsMap := make(map[string]expectedPermission, len(expectedPermissions))
	for _, expectedPermission := range expectedPermissions {
		expectedPermissionsMap[getResourceAccessKey(expectedPermission.ResourceAccess)] = expectedPermission
	}

	var missing []expectedPermission
	var redundant []clientmodels.ResourceAccess

	// Verify all expected permissions are present.
	for _, permission := range expectedPermissions {
		if _, ok := actualRequiredResources[getResourceAccessKey(permission.ResourceAccess)]; !ok {
			missing = append(missing, permission)
		}
	}

	// Check for unnecessary permissions.
	for _, requiredResource := range app.RequiredResources {
		if _, ok := expectedPermissionsMap[getResourceAccessKey(requiredResource)]; !ok {
			redundant = append(redundant, requiredResource)
		}
	}

	return missing, redundant
}
