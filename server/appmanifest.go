// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
)

const (
	AppVersion          = "0.1.0"
	AppID               = "3dfcc488-f38d-4867-bcba-84dfc8eb007b"
	PackageName         = "com.mattermost.msteams.devsecops"
	TabAppID            = "85ffd111-223c-47f8-a9ea-7568f2012f65"
	TabAppURI           = "api://%s/plugins/" + pluginID + "/iframe/" + TabAppID
	ManifestName        = "manifest.json"
	LogoColorFilename   = "mm-logo-color.png"
	LogoOutlineFilename = "mm-logo-outline.png"
)

// iFrameManifest returns the Mattermost for MS Teams app manifest as a zip file.
// This zip file can be imported as a MS Teams app.
func (a *API) iFrameManifest(w http.ResponseWriter, _ *http.Request) {
	config := a.p.API.GetConfig()
	siteURL := *config.ServiceSettings.SiteURL
	if siteURL == "" {
		a.p.API.LogError("SiteURL cannot be empty for MS Teams app manifest")
		http.Error(w, "SiteURL is empty", http.StatusInternalServerError)
		return
	}

	publicHostName, protocol, err := parseDomain(siteURL)
	if err != nil {
		a.p.API.LogError("SiteURL is invalid for MS Teams app manifest", "error", err.Error())
		http.Error(w, "SiteURL is invalid: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tabURI := fmt.Sprintf(TabAppURI, publicHostName)

	manifest := strings.ReplaceAll(manifestJSON, "{{VERSION}}", AppVersion)
	manifest = strings.ReplaceAll(manifest, "{{APP_ID}}", AppID)
	manifest = strings.ReplaceAll(manifest, "{{PACKAGE_NAME}}", PackageName)
	manifest = strings.ReplaceAll(manifest, "{{PROTOCOL}}", protocol)
	manifest = strings.ReplaceAll(manifest, "{{PUBLIC_HOSTNAME}}", publicHostName)
	manifest = strings.ReplaceAll(manifest, "{{TAB_APP_ID}}", TabAppID)
	manifest = strings.ReplaceAll(manifest, "{{TAB_APP_URI}}", tabURI)
	manifest = strings.ReplaceAll(manifest, "{{PLUGIN_ID}}", pluginID)

	bufReader, err := createManifestZip(
		zipFile{name: ManifestName, data: []byte(manifest)},
		zipFile{name: LogoColorFilename, data: assets.LogoColorData},
		zipFile{name: LogoOutlineFilename, data: assets.LogoOutlineData},
	)
	if err != nil {
		a.p.API.LogWarn("Error generating app manifest", "error", err.Error())
		http.Error(w, "Error generating app manifest", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=com.mattermost.msteamsapp.zip")

	if _, err := io.Copy(w, bufReader); err != nil {
		a.p.API.LogWarn("Unable to serve the app manifest", "error", err.Error())
	}
}

func parseDomain(uri string) (string, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}
	return u.Host, u.Scheme, nil
}

type zipFile struct {
	name string
	data []byte
}

func createManifestZip(files ...zipFile) (io.Reader, error) {
	buf := &bytes.Buffer{}

	w := zip.NewWriter(buf)
	defer w.Close()

	for _, zf := range files {
		fw, err := w.Create(zf.name)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(fw, bytes.NewReader(zf.data)); err != nil {
			return nil, err
		}
	}

	return buf, nil
}

var manifestJSON = `{
	"$schema": "https://developer.microsoft.com/en-us/json-schemas/teams/v1.15/MicrosoftTeams.schema.json",
	"manifestVersion": "1.15",
	"id": "{{APP_ID}}",
	"version": "{{VERSION}}",
	"packageName": "{{PACKAGE_NAME}}",
	"developer": {
	  "name": "Mattermost",
	  "websiteUrl": "https://github.com/mattermost/mattermost-plugin-msteams-devsecops",
	  "privacyUrl": "https://mattermost.com/privacy-policy/",
	  "termsOfUseUrl": "https://mattermost.com/terms-of-use/"
	},
	"name": {
	  "short": "Mattermost for MS Teams",
	  "full": "Mattermost app for Microsoft Teams"
	},
	"description": {
	  "short": "Mattermost for MS Teams",
	  "full": "Mattermost app for Microsoft Teams"
	},
	"icons": {
	  "outline": "mm-logo-outline.png",
	  "color": "mm-logo-color.png"
	},
	"accentColor": "#FFFFFF",
	"configurableTabs": [],
	"staticTabs": [
	  {
		"entityId": "f607c5e9-7175-44ee-ba14-10e33a7b4c91",
		"name": "Mattermost",
		"contentUrl": "{{PROTOCOL}}://{{PUBLIC_HOSTNAME}}/plugins/{{PLUGIN_ID}}/iframe/mattermostTab?name={loginHint}&tenant={tid}&theme={theme}",
		"scopes": [
		  "personal"
		]
	  }
	],
	"bots": [],
	"connectors": [],
	"composeExtensions": [],
	"permissions": [
	  "identity",
	  "messageTeamMembers"
	],
	"validDomains": [
	  "{{PUBLIC_HOSTNAME}}"
	],
	"showLoadingIndicator": false,
	"isFullScreen": true,
	"webApplicationInfo": {
	  "id": "{{TAB_APP_ID}}",
	  "resource": "{{TAB_APP_URI}}"
	}
  }
`
