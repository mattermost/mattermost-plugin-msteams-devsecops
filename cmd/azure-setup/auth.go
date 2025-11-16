// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
)

// authenticateToAzure establishes authentication to Azure using available methods
// Priority order: Environment variables -> Azure CLI -> Interactive browser
func authenticateToAzure(ctx context.Context, tenantID string, verbose bool) (azcore.TokenCredential, error) {
	if verbose {
		fmt.Println("🔐 Authenticating to Azure...")
	}

	// Try multiple authentication methods in order of preference
	credential, method, err := tryAuthenticationMethods(ctx, tenantID, verbose)
	if err != nil {
		return nil, errors.Wrap(err, "failed to authenticate to Azure")
	}

	if verbose {
		fmt.Printf("✅ Successfully authenticated using: %s\n", method)
	}

	return credential, nil
}

// tryAuthenticationMethods attempts different authentication methods
func tryAuthenticationMethods(ctx context.Context, tenantID string, verbose bool) (azcore.TokenCredential, string, error) {
	// Method 1: Try environment variables (service principal)
	if verbose {
		fmt.Println("   Trying: Environment variables (Service Principal)...")
	}
	if cred, err := tryEnvironmentCredential(tenantID); err == nil {
		if err := testCredential(ctx, cred); err == nil {
			return cred, "Environment Variables (Service Principal)", nil
		}
	}

	// Method 2: Try Azure CLI
	if verbose {
		fmt.Println("   Trying: Azure CLI...")
	}
	if cred, err := tryAzureCLICredential(tenantID); err == nil {
		if err := testCredential(ctx, cred); err == nil {
			return cred, "Azure CLI", nil
		}
	}

	// Method 3: Try interactive browser
	if verbose {
		fmt.Println("   Trying: Interactive browser...")
	}
	if cred, err := tryInteractiveBrowserCredential(tenantID); err == nil {
		if err := testCredential(ctx, cred); err == nil {
			return cred, "Interactive Browser", nil
		}
	}

	// Method 4: Device code flow (last resort, works on headless systems)
	if verbose {
		fmt.Println("   Trying: Device code flow...")
	}
	if cred, err := tryDeviceCodeCredential(tenantID); err == nil {
		if err := testCredential(ctx, cred); err == nil {
			return cred, "Device Code Flow", nil
		}
	}

	return nil, "", errors.New("all authentication methods failed")
}

// tryEnvironmentCredential attempts to authenticate using environment variables
func tryEnvironmentCredential(tenantID string) (azcore.TokenCredential, error) {
	opts := &azidentity.ClientSecretCredentialOptions{}
	if tenantID != "" {
		opts.AdditionallyAllowedTenants = []string{tenantID}
	}

	cred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{
		ClientOptions: opts.ClientOptions,
	})
	if err != nil {
		return nil, err
	}

	return cred, nil
}

// tryAzureCLICredential attempts to authenticate using Azure CLI
func tryAzureCLICredential(tenantID string) (azcore.TokenCredential, error) {
	opts := &azidentity.AzureCLICredentialOptions{}
	if tenantID != "" {
		opts.TenantID = tenantID
	}

	cred, err := azidentity.NewAzureCLICredential(opts)
	if err != nil {
		return nil, err
	}

	return cred, nil
}

// tryInteractiveBrowserCredential attempts to authenticate using interactive browser
func tryInteractiveBrowserCredential(tenantID string) (azcore.TokenCredential, error) {
	opts := &azidentity.InteractiveBrowserCredentialOptions{}
	if tenantID != "" {
		opts.TenantID = tenantID
	}

	cred, err := azidentity.NewInteractiveBrowserCredential(opts)
	if err != nil {
		return nil, err
	}

	return cred, nil
}

// tryDeviceCodeCredential attempts to authenticate using device code flow
func tryDeviceCodeCredential(tenantID string) (azcore.TokenCredential, error) {
	opts := &azidentity.DeviceCodeCredentialOptions{
		UserPrompt: func(ctx context.Context, message azidentity.DeviceCodeMessage) error {
			fmt.Println("\n" + message.Message)
			return nil
		},
	}
	if tenantID != "" {
		opts.TenantID = tenantID
	}

	cred, err := azidentity.NewDeviceCodeCredential(opts)
	if err != nil {
		return nil, err
	}

	return cred, nil
}

// testCredential verifies that a credential can obtain a token
func testCredential(ctx context.Context, cred azcore.TokenCredential) error {
	// Try to get a token for Microsoft Graph
	_, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	return err
}

// validateAzureConnection ensures we can connect to Azure and get basic info
func validateAzureConnection(ctx context.Context, cred azcore.TokenCredential, verbose bool) error {
	if verbose {
		fmt.Println("🔍 Validating Azure connection...")
	}

	// Try to get a token to verify the connection works
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return errors.Wrap(err, "failed to obtain access token from Azure")
	}

	if token.Token == "" {
		return errors.New("received empty access token from Azure")
	}

	if verbose {
		fmt.Println("✅ Azure connection validated")
	}

	return nil
}
