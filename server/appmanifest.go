// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
)

const (
	PackageName         = "com.mattermost.msteams.devsecops"
	ManifestName        = "manifest.json"
	LogoColorFilename   = "mm-logo-color.png"
	LogoOutlineFilename = "mm-logo-outline.png"
)

type manifestContext struct {
	AppVersion     string // the app version
	AppPackageName string // fully qualified package name for the app (e.g. com.mattermost.msteams.devsecops)
	AppID          string // the unique app id
	AppClientID    string // the app's client ID as defined in Azure portal
	AppName        string // short and full name of the app

	SiteDomain     string // the domain name extracted from this Mattermost server's site url. No protocol or path.
	SiteDomainPath string // the domain name and path extracted from this Mattermost server's site url. No protocol.
	PluginID       string // the plugin ID (e.g. com.mattermost.msteams.devsecops)
}

// makeManifestContext populates a manifestContext with template data
func (a *API) makeManifestContext() (*manifestContext, error) {
	config := a.p.API.GetConfig()
	siteURL := *config.ServiceSettings.SiteURL
	if siteURL == "" {
		return nil, errors.New("SiteURL cannot be empty")
	}

	_, hostName, path, err := parseURL(siteURL)
	if err != nil {
		return nil, fmt.Errorf("SiteURL is invalid: %w", err)
	}

	pluginConfig := a.p.getConfiguration()
	if err := a.p.validateConfiguration(pluginConfig); err != nil {
		return nil, fmt.Errorf("plugin configuration is invalid: %w", err)
	}

	return &manifestContext{
		AppVersion:     pluginConfig.AppVersion,
		AppPackageName: PackageName,
		AppID:          pluginConfig.AppID,
		AppClientID:    pluginConfig.AppClientID,
		AppName:        pluginConfig.AppName,
		SiteDomain:     hostName,
		SiteDomainPath: strings.TrimRight(strings.Join([]string{hostName, path}, "/"), "/"),
		PluginID:       a.p.API.GetPluginID(),
	}, nil
}

// appManifest returns the Mattermost for MS Teams app manifest as a zip file.
// This zip file can be imported as a MS Teams tab app.
func (a *API) appManifest(w http.ResponseWriter, _ *http.Request) {
	tmplContext, err := a.makeManifestContext()
	if err != nil {
		a.p.API.LogError("Unable to create app manifest context", "error", err.Error())
		http.Error(w, "Unable to create app manifest context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("manifest").Parse(assets.AppManifestTemplate)
	if err != nil {
		a.p.API.LogError("Unable to parse app manifest template", "error", err.Error())
		http.Error(w, "Unable to parse app manifest template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, tmplContext); err != nil {
		a.p.API.LogError("Unable to execute app manifest template", "error", err.Error())
		http.Error(w, "Unable to execute app manifest template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a zip file with the manifest and logo files
	bufReader, err := createManifestZip(
		zipFile{name: ManifestName, data: buf.Bytes()},
		zipFile{name: LogoColorFilename, data: assets.LogoColorData},
		zipFile{name: LogoOutlineFilename, data: assets.LogoOutlineData},
	)
	if err != nil {
		a.p.API.LogError("Error generating app manifest", "error", err.Error())
		http.Error(w, "Error generating app manifest"+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s-%s.zip", PackageName, tmplContext.AppVersion)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	if _, err := io.Copy(w, bufReader); err != nil {
		a.p.API.LogWarn("Unable to serve the app manifest", "error", err.Error())
	}
}

func parseURL(uri string) (string, string, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", "", err
	}
	return u.Scheme, u.Host, u.Path, nil
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
