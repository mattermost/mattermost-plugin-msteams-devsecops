// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name        string
		config      *SetupConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  12,
			},
			expectError: false,
		},
		{
			name: "missing site URL",
			config: &SetupConfig{
				AppName:          "Mattermost for Teams",
				SecretExpiration: 12,
			},
			expectError: true,
			errorMsg:    "Mattermost site URL is required",
		},
		{
			name: "invalid site URL - no protocol",
			config: &SetupConfig{
				MattermostSiteURL: "mattermost.example.com",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  12,
			},
			expectError: true,
			errorMsg:    "must include protocol",
		},
		{
			name: "invalid site URL - not HTTPS",
			config: &SetupConfig{
				MattermostSiteURL: "http://mattermost.example.com",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  12,
			},
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
		{
			name: "missing app name",
			config: &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				SecretExpiration:  12,
			},
			expectError: true,
			errorMsg:    "application name is required",
		},
		{
			name: "app name too short",
			config: &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				AppName:           "MM",
				SecretExpiration:  12,
			},
			expectError: true,
			errorMsg:    "must be at least 3 characters",
		},
		{
			name: "secret expiration too long",
			config: &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  25,
			},
			expectError: true,
			errorMsg:    "cannot exceed 24 months",
		},
		{
			name: "default secret expiration when zero",
			config: &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  0,
			},
			expectError: false,
		},
		{
			name: "site URL with path",
			config: &SetupConfig{
				MattermostSiteURL: "https://example.com/mattermost",
				AppName:           "Mattermost for Teams",
				SecretExpiration:  12,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInputs(tt.config)

			if tt.expectError {
				require.Error(t, err, "expected an error but got none")
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, "expected no error but got: %v", err)
				// Verify default secret expiration is set
				if tt.config.SecretExpiration == 0 {
					assert.Equal(t, 12, tt.config.SecretExpiration, "default secret expiration should be 12 months")
				}
			}
		})
	}
}

func TestExtractHostnameAndPath(t *testing.T) {
	tests := []struct {
		name             string
		siteURL          string
		expectedHostname string
		expectedPath     string
		expectError      bool
	}{
		{
			name:             "simple domain",
			siteURL:          "https://mattermost.example.com",
			expectedHostname: "mattermost.example.com",
			expectedPath:     "",
			expectError:      false,
		},
		{
			name:             "domain with port",
			siteURL:          "https://mattermost.example.com:8065",
			expectedHostname: "mattermost.example.com:8065",
			expectedPath:     "",
			expectError:      false,
		},
		{
			name:             "domain with path",
			siteURL:          "https://example.com/mattermost",
			expectedHostname: "example.com",
			expectedPath:     "/mattermost",
			expectError:      false,
		},
		{
			name:             "domain with nested path",
			siteURL:          "https://example.com/apps/mattermost",
			expectedHostname: "example.com",
			expectedPath:     "/apps/mattermost",
			expectError:      false,
		},
		{
			name:             "domain with trailing slash",
			siteURL:          "https://mattermost.example.com/",
			expectedHostname: "mattermost.example.com",
			expectedPath:     "",
			expectError:      false,
		},
		{
			name:        "invalid URL - contains invalid characters",
			siteURL:     "https://[invalid url with spaces]",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname, path, err := extractHostnameAndPath(tt.siteURL)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedHostname, hostname)
				assert.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

func TestBuildApplicationIDURI(t *testing.T) {
	tests := []struct {
		name        string
		siteURL     string
		clientID    string
		expected    string
		expectError bool
	}{
		{
			name:        "simple domain",
			siteURL:     "https://mattermost.example.com",
			clientID:    "abc123-def456",
			expected:    "api://mattermost.example.com/abc123-def456",
			expectError: false,
		},
		{
			name:        "domain with port",
			siteURL:     "https://mattermost.example.com:8065",
			clientID:    "abc123-def456",
			expected:    "api://mattermost.example.com:8065/abc123-def456",
			expectError: false,
		},
		{
			name:        "domain with path",
			siteURL:     "https://example.com/mattermost",
			clientID:    "abc123-def456",
			expected:    "api://example.com/mattermost/abc123-def456",
			expectError: false,
		},
		{
			name:        "domain with nested path",
			siteURL:     "https://example.com/apps/mattermost",
			clientID:    "abc123-def456",
			expected:    "api://example.com/apps/mattermost/abc123-def456",
			expectError: false,
		},
		{
			name:        "domain with trailing slash",
			siteURL:     "https://mattermost.example.com/",
			clientID:    "abc123-def456",
			expected:    "api://mattermost.example.com/abc123-def456",
			expectError: false,
		},
		{
			name:        "invalid URL - contains invalid characters",
			siteURL:     "https://[invalid url with spaces]",
			clientID:    "abc123-def456",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildApplicationIDURI(tt.siteURL, tt.clientID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsApplicationAdminRole(t *testing.T) {
	tests := []struct {
		name           string
		roleTemplateID string
		expected       bool
	}{
		{
			name:           "Application Administrator",
			roleTemplateID: "9b895d92-2cd3-44c7-9d02-a6ac2d5ea5c3",
			expected:       true,
		},
		{
			name:           "Cloud Application Administrator",
			roleTemplateID: "158c047a-c907-4556-b7ef-446551a6b5f7",
			expected:       true,
		},
		{
			name:           "Global Administrator",
			roleTemplateID: "62e90394-69f5-4237-9190-012177145e10",
			expected:       true,
		},
		{
			name:           "Unknown role",
			roleTemplateID: "00000000-0000-0000-0000-000000000000",
			expected:       false,
		},
		{
			name:           "User role",
			roleTemplateID: "a0b1b346-4d3e-4e8b-98f8-753987be4970",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isApplicationAdminRole(tt.roleTemplateID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeODataString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "Mattermost for Teams",
			expected: "Mattermost for Teams",
		},
		{
			name:     "single quote",
			input:    "O'Brien's App",
			expected: "O''Brien''s App",
		},
		{
			name:     "multiple single quotes",
			input:    "It's Mike's App",
			expected: "It''s Mike''s App",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only single quote",
			input:    "'",
			expected: "''",
		},
		{
			name:     "UUID (no escaping needed)",
			input:    "abc123-def456-789",
			expected: "abc123-def456-789",
		},
		{
			name:     "special characters other than single quote",
			input:    "App-Name_123 (Test)",
			expected: "App-Name_123 (Test)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeODataString(tt.input)
			assert.Equal(t, tt.expected, result, "OData escaping should handle single quotes")
		})
	}
}
