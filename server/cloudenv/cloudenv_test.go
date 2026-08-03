// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cloudenv

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentForResolution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
	}{
		{"empty defaults to commercial", "", Commercial},
		{"unknown defaults to commercial", "azure-secret", Commercial},
		{"commercial", "commercial", Commercial},
		{"gcchigh", "gcchigh", GCCHigh},
		{"dod", "dod", DoD},
		{"case insensitive", "GCCHigh", GCCHigh},
		{"trims whitespace", "  dod  ", DoD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, EnvironmentFor(tt.input).Name)
		})
	}
}

// TestCommercialByteIdentical guards against regressions: the commercial
// environment must produce exactly the strings the plugin hardcoded before
// national cloud support was added.
func TestCommercialByteIdentical(t *testing.T) {
	env := EnvironmentFor(Commercial)

	assert.Equal(t, "login.microsoftonline.com", env.LoginAuthorityHost)
	assert.Equal(t, "graph.microsoft.com", env.GraphHost)
	assert.Equal(t, "https://graph.microsoft.com/v1.0", env.GraphBaseURL)
	assert.Equal(t, "https://graph.microsoft.com/.default", env.GraphScope)
	assert.Equal(t, "https://login.microsoftonline.com/common/discovery/v2.0/keys", env.JWKSURL)
	assert.Equal(t, "teams.microsoft.com", env.TeamsDomain)
	assert.Equal(t, "portal.azure.com", env.PortalHost)

	// The previous hardcoded frame-ancestors string (server/plugin.go).
	assert.Equal(t,
		"*.cloud.microsoft teams.microsoft.com *.teams.microsoft.com *.microsoft365.com *.office.com outlook.office.com outlook.office365.com outlook-sdf.office.com outlook-sdf.office365.com",
		strings.Join(env.FrameAncestors, " "),
	)

	// The previous hardcoded DefaultCSPConnectSrc (server/api_csp.go).
	assert.Equal(t, "https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net", env.CSPConnectSrc)

	// Commercial must use the Azure public cloud authority.
	assert.Equal(t, cloud.AzurePublic.ActiveDirectoryAuthorityHost, env.AzureCloud.ActiveDirectoryAuthorityHost)
}

func TestGCCHighEndpoints(t *testing.T) {
	env := EnvironmentFor(GCCHigh)

	assert.Equal(t, "login.microsoftonline.us", env.LoginAuthorityHost)
	assert.Equal(t, "graph.microsoft.us", env.GraphHost)
	assert.Equal(t, "https://graph.microsoft.us/v1.0", env.GraphBaseURL)
	assert.Equal(t, "https://graph.microsoft.us/.default", env.GraphScope)
	assert.Equal(t, "https://login.microsoftonline.us/common/discovery/v2.0/keys", env.JWKSURL)
	assert.Equal(t, "gov.teams.microsoft.us", env.TeamsDomain)
	assert.Equal(t, "portal.azure.us", env.PortalHost)
	assert.Equal(t, cloud.AzureGovernment.ActiveDirectoryAuthorityHost, env.AzureCloud.ActiveDirectoryAuthorityHost)
	assert.Contains(t, env.FrameAncestors, "gov.teams.microsoft.us")
	assert.Contains(t, env.CSPConnectSrc, "https://*.teams.microsoft.us")
}

func TestDoDEndpoints(t *testing.T) {
	env := EnvironmentFor(DoD)

	// DoD shares the login authority with GCC High but uses a distinct Graph host.
	assert.Equal(t, "login.microsoftonline.us", env.LoginAuthorityHost)
	assert.Equal(t, "dod-graph.microsoft.us", env.GraphHost)
	assert.Equal(t, "https://dod-graph.microsoft.us/v1.0", env.GraphBaseURL)
	assert.Equal(t, "https://dod-graph.microsoft.us/.default", env.GraphScope)
	assert.Equal(t, "https://login.microsoftonline.us/common/discovery/v2.0/keys", env.JWKSURL)
	assert.Equal(t, "dod.teams.microsoft.us", env.TeamsDomain)
	assert.Equal(t, "portal.azure.us", env.PortalHost)
	assert.Equal(t, cloud.AzureGovernment.ActiveDirectoryAuthorityHost, env.AzureCloud.ActiveDirectoryAuthorityHost)
	assert.Contains(t, env.FrameAncestors, "dod.teams.microsoft.us")
}

func TestNames(t *testing.T) {
	assert.Equal(t, []string{Commercial, GCCHigh, DoD}, Names())
}
