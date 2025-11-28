// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
)

// showPreflightConfirmation displays a summary of planned changes and prompts for confirmation
func showPreflightConfirmation(config *SetupConfig, existingApp models.Applicationable) error {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🔍 PRE-FLIGHT CHECK")
	fmt.Println(strings.Repeat("=", 70))

	if existingApp != nil {
		fmt.Println("\n📝 Action: Update existing Azure application")
		fmt.Printf("   Application Name: %s\n", *existingApp.GetDisplayName())
		fmt.Printf("   Application ID:   %s\n", *existingApp.GetAppId())
	} else {
		fmt.Println("\n🆕 Action: Create new Azure application")
		fmt.Printf("   Application Name: %s\n", config.AppName)
	}

	fmt.Println("\n📋 Configuration Summary:")
	fmt.Printf("   Mattermost Site URL:    %s\n", config.MattermostSiteURL)
	fmt.Printf("   Secret Expiration:      %d months\n", config.SecretExpiration)

	fmt.Println("\n🔐 API Permissions to configure:")
	for _, perm := range getRequiredPermissions() {
		permType := "Delegated"
		if perm.Type == PermissionTypeRole {
			permType = "Application"
		}
		fmt.Printf("   • %s (%s)\n", perm.Name, permType)
	}

	fmt.Println("\n🌐 API Exposure:")
	appIDURI, _ := buildApplicationIDURI(config.MattermostSiteURL, "CLIENT_ID")
	appIDURI = strings.Replace(appIDURI, "CLIENT_ID", "{client-id}", 1)
	fmt.Printf("   • Application ID URI: %s\n", appIDURI)
	fmt.Printf("   • Scope: %s\n", ScopeName)
	fmt.Printf("   • Pre-authorized clients: %d Microsoft apps (Teams, Outlook, Office, Copilot)\n", len(getPreAuthorizedClients()))

	fmt.Println("\n🔑 Operations to perform:")
	if existingApp == nil {
		fmt.Println("   1. Create new Azure AD application")
	} else {
		fmt.Println("   1. Update existing Azure AD application")
	}
	fmt.Println("   2. Configure API permissions")
	fmt.Println("   3. Set up API exposure and scopes")
	fmt.Println("   4. Add pre-authorized Microsoft clients")
	fmt.Println("   5. Generate new client secret")
	fmt.Println("   6. Create service principal (if needed)")

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Print("\nProceed with these changes? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "failed to read user input")
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("\n❌ Operation cancelled by user")
		return errors.New("operation cancelled by user")
	}

	fmt.Println()
	return nil
}
