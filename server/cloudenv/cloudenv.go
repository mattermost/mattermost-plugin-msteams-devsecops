// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package cloudenv describes the Microsoft national cloud environments this plugin
// supports (commercial, US Government GCC High, and US Government DoD) and maps each
// to the endpoints, scopes, and domains that differ between clouds.
//
// GCC (moderate) is intentionally not a separate environment: it rides on the
// commercial endpoints and is therefore covered by Commercial. China (21Vianet) and
// air-gapped classified clouds are out of scope.
//
// Note: cloud.AzureGovernment selects the login authority (login.microsoftonline.us)
// for both GCC High and DoD, but does not distinguish their Graph hosts
// (graph.microsoft.us vs dod-graph.microsoft.us), so GraphBaseURL is set explicitly.
package cloudenv

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
)

// Recognized cloud names. These are the accepted values for the plugin's
// national_cloud setting and the azure-setup --cloud flag.
const (
	Commercial = "commercial"
	GCCHigh    = "gcchigh"
	DoD        = "dod"
)

// Environment holds all cloud-specific endpoints and domains for a Microsoft
// national cloud.
type Environment struct {
	// Name is the recognized cloud name (see the constants above).
	Name string

	// LoginAuthorityHost is the Microsoft Entra host used to build OAuth
	// authorize/token URLs (e.g. login.microsoftonline.com).
	LoginAuthorityHost string

	// GraphHost is the Microsoft Graph service host (e.g. graph.microsoft.com).
	GraphHost string

	// GraphBaseURL is the Microsoft Graph service root, including the version
	// segment (e.g. https://graph.microsoft.com/v1.0). It is set explicitly on
	// the Graph SDK adapter because DoD uses a different Graph host than GCC High.
	GraphBaseURL string

	// GraphScope is the OAuth .default scope for Microsoft Graph
	// (e.g. https://graph.microsoft.com/.default).
	GraphScope string

	// JWKSURL serves the signing keys used to validate Teams SSO tokens.
	JWKSURL string

	// AzureCloud is the azcore cloud configuration passed to the Azure SDK
	// credentials so tokens are acquired from the correct authority.
	AzureCloud cloud.Configuration

	// TeamsDomain is the Microsoft Teams client host, used for activity-feed
	// deep links and for embedding (e.g. teams.microsoft.com).
	TeamsDomain string

	// PortalHost is the Azure portal host, used for admin-consent links
	// (e.g. portal.azure.com).
	PortalHost string

	// FrameAncestors lists the domains added to the Mattermost server's
	// frame-ancestors so the plugin iframe can be embedded in this cloud's
	// Microsoft 365 clients.
	//
	// The Teams client domain is authoritative. The Outlook/M365 gov domains
	// below are best-effort and may need refinement once verified against a
	// gov tenant (tracked as a verification item in the implementation plan).
	FrameAncestors []string

	// CSPConnectSrc is the connect-src value for the iframe Content Security
	// Policy. The script-src is intentionally not per-cloud: the Teams JS SDK
	// is served from the same res.cdn.office.net CDN across all clouds.
	CSPConnectSrc string
}

func commercial() Environment {
	return Environment{
		Name:               Commercial,
		LoginAuthorityHost: "login.microsoftonline.com",
		GraphHost:          "graph.microsoft.com",
		GraphBaseURL:       "https://graph.microsoft.com/v1.0",
		GraphScope:         "https://graph.microsoft.com/.default",
		JWKSURL:            "https://login.microsoftonline.com/common/discovery/v2.0/keys",
		AzureCloud:         cloud.AzurePublic,
		TeamsDomain:        "teams.microsoft.com",
		PortalHost:         "portal.azure.com",
		FrameAncestors: []string{
			"*.cloud.microsoft",
			"teams.microsoft.com",
			"*.teams.microsoft.com",
			"*.microsoft365.com",
			"*.office.com",
			"outlook.office.com",
			"outlook.office365.com",
			"outlook-sdf.office.com",
			"outlook-sdf.office365.com",
		},
		CSPConnectSrc: "https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net",
	}
}

func gccHigh() Environment {
	return Environment{
		Name:               GCCHigh,
		LoginAuthorityHost: "login.microsoftonline.us",
		GraphHost:          "graph.microsoft.us",
		GraphBaseURL:       "https://graph.microsoft.us/v1.0",
		GraphScope:         "https://graph.microsoft.us/.default",
		JWKSURL:            "https://login.microsoftonline.us/common/discovery/v2.0/keys",
		AzureCloud:         cloud.AzureGovernment,
		TeamsDomain:        "gov.teams.microsoft.us",
		PortalHost:         "portal.azure.us",
		FrameAncestors: []string{
			"*.cloud.microsoft",
			"gov.teams.microsoft.us",
			"*.gov.teams.microsoft.us",
			"*.office365.us",
			"outlook.office365.us",
		},
		CSPConnectSrc: "https://*.microsoft.us https://*.teams.microsoft.us https://*.cdn.office.net",
	}
}

func dod() Environment {
	return Environment{
		Name:               DoD,
		LoginAuthorityHost: "login.microsoftonline.us",
		GraphHost:          "dod-graph.microsoft.us",
		GraphBaseURL:       "https://dod-graph.microsoft.us/v1.0",
		GraphScope:         "https://dod-graph.microsoft.us/.default",
		JWKSURL:            "https://login.microsoftonline.us/common/discovery/v2.0/keys",
		AzureCloud:         cloud.AzureGovernment,
		TeamsDomain:        "dod.teams.microsoft.us",
		PortalHost:         "portal.azure.us",
		FrameAncestors: []string{
			"*.cloud.microsoft",
			"dod.teams.microsoft.us",
			"*.dod.teams.microsoft.us",
			"*.dod.online.office365.us",
		},
		CSPConnectSrc: "https://*.microsoft.us https://*.teams.microsoft.us https://*.cdn.office.net",
	}
}

// EnvironmentFor returns the cloud environment for the given name, defaulting to
// the commercial cloud for empty or unrecognized names. Matching is
// case-insensitive and ignores surrounding whitespace.
func EnvironmentFor(name string) Environment {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case GCCHigh:
		return gccHigh()
	case DoD:
		return dod()
	default:
		return commercial()
	}
}

// Names returns the recognized cloud names, commercial first.
func Names() []string {
	return []string{Commercial, GCCHigh, DoD}
}
