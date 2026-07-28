// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"testing"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-msteams-embedded/server/cloudenv"
)

// TestEnsureJWKSRebuildsOnCloudChange is the Issue 1 regression test: changing the
// national_cloud setting must switch the JWKS URL the keyfunc validates against.
// Before the fix the keyfunc was built once on first activation and never rebuilt
// on a config change, so a gov tenant's tokens were validated against the
// commercial JWKS and SSO broke.
func TestEnsureJWKSRebuildsOnCloudChange(t *testing.T) {
	var builtURLs []string
	var cancels int

	orig := jwksSetup
	jwksSetup = func(url string) (keyfunc.Keyfunc, context.CancelFunc) {
		builtURLs = append(builtURLs, url)
		return nil, func() { cancels++ }
	}
	defer func() { jwksSetup = orig }()

	commercialURL := cloudenv.EnvironmentFor(cloudenv.Commercial).JWKSURL
	gccHighURL := cloudenv.EnvironmentFor(cloudenv.GCCHigh).JWKSURL
	require.NotEqual(t, commercialURL, gccHighURL)

	p := &Plugin{}

	// First activation on the commercial default builds the keyfunc once.
	p.setConfiguration(&configuration{NationalCloud: cloudenv.Commercial})
	p.ensureJWKS()
	assert.Equal(t, []string{commercialURL}, builtURLs)
	assert.Equal(t, commercialURL, p.activeJWKSURL)
	assert.Equal(t, 0, cancels)

	// No cloud change: nothing is rebuilt.
	p.ensureJWKS()
	assert.Equal(t, []string{commercialURL}, builtURLs)
	assert.Equal(t, 0, cancels)

	// Switching to GCC High rebuilds against the gov JWKS and tears down the old.
	p.setConfiguration(&configuration{NationalCloud: cloudenv.GCCHigh})
	p.ensureJWKS()
	assert.Equal(t, []string{commercialURL, gccHighURL}, builtURLs)
	assert.Equal(t, gccHighURL, p.activeJWKSURL)
	assert.Equal(t, 1, cancels)
}

// TestStartSkipsWhenUnconfigured verifies that starting the plugin without M365
// credentials logs a single informational message and returns without connecting
// or scheduling the credentials job (so an unconfigured install stays quiet and
// does not panic).
func TestStartSkipsWhenUnconfigured(t *testing.T) {
	orig := jwksSetup
	jwksSetup = func(string) (keyfunc.Keyfunc, context.CancelFunc) { return nil, func() {} }
	defer func() { jwksSetup = orig }()

	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything).Return()

	p := &Plugin{}
	p.API = api
	p.setConfiguration(&configuration{})

	assert.NotPanics(t, func() { p.start(false) })

	api.AssertCalled(t, "LogInfo", mock.Anything)
	assert.Nil(t, p.GetClientForApp(), "no Teams client should be created when unconfigured")
	api.AssertNotCalled(t, "LogError", mock.Anything, mock.Anything, mock.Anything)
}

// TestConnectTeamsAppClientClearsClientOnFailure verifies that a failed Connect()
// leaves no cached client, so GetClientForApp reports nil and the credentials
// check skips cleanly instead of dereferencing a half-initialized client.
func TestConnectTeamsAppClientClearsClientOnFailure(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	p := &Plugin{}
	p.API = api
	p.client = pluginapi.NewClient(api, nil)
	// Tenant and client ID set but secret empty, so Connect() fails building the
	// client secret credential.
	p.setConfiguration(&configuration{M365TenantID: "tenant", M365ClientID: "client"})

	err := p.connectTeamsAppClient()
	require.Error(t, err)
	assert.Nil(t, p.GetClientForApp(), "client should be cleared after a failed Connect")
}
