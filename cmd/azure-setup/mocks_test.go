// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/google/uuid"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

// Mock helper functions to create test data without hitting Azure

// newMockApplication creates a mock Azure AD application for testing
func newMockApplication(clientID, objectID, displayName string) models.Applicationable {
	app := models.NewApplication()
	app.SetAppId(&clientID)
	app.SetId(&objectID)
	app.SetDisplayName(&displayName)

	signInAudience := "AzureADMyOrg"
	app.SetSignInAudience(&signInAudience)

	return app
}

// newMockServicePrincipal creates a mock service principal for testing
func newMockServicePrincipal(appID, objectID string) models.ServicePrincipalable {
	sp := models.NewServicePrincipal()
	sp.SetAppId(&appID)
	sp.SetId(&objectID)

	return sp
}

// newMockPasswordCredential creates a mock password credential for testing
func newMockPasswordCredential(secretValue string) models.PasswordCredentialable {
	cred := models.NewPasswordCredential()
	cred.SetSecretText(&secretValue)

	keyID := uuid.New()
	cred.SetKeyId(&keyID)

	displayName := "Test Secret"
	cred.SetDisplayName(&displayName)

	return cred
}

// newMockApplicationCollection creates a mock application collection for testing
func newMockApplicationCollection(apps []models.Applicationable) models.ApplicationCollectionResponseable {
	collection := models.NewApplicationCollectionResponse()
	collection.SetValue(apps)
	return collection
}

// newMockServicePrincipalCollection creates a mock service principal collection
func newMockServicePrincipalCollection(sps []models.ServicePrincipalable) models.ServicePrincipalCollectionResponseable {
	collection := models.NewServicePrincipalCollectionResponse()
	collection.SetValue(sps)
	return collection
}

// newMockOrganization creates a mock organization for testing
func newMockOrganization(tenantID string) models.OrganizationCollectionResponseable {
	org := models.NewOrganization()
	org.SetId(&tenantID)

	collection := models.NewOrganizationCollectionResponse()
	collection.SetValue([]models.Organizationable{org})

	return collection
}

// newMockUser creates a mock user for testing
func newMockUser(upn string) models.Userable {
	user := models.NewUser()
	user.SetUserPrincipalName(&upn)

	userID := uuid.New().String()
	user.SetId(&userID)

	return user
}

// newMockDirectoryRole creates a mock directory role for testing
func newMockDirectoryRole(roleTemplateID, displayName string) models.DirectoryRoleable {
	role := models.NewDirectoryRole()
	role.SetRoleTemplateId(&roleTemplateID)
	role.SetDisplayName(&displayName)

	roleID := uuid.New().String()
	role.SetId(&roleID)

	return role
}
