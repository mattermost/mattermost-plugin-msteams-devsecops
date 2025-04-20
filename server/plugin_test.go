// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/v8/channels/utils"
)

func getURL(config *model.Config) string {
	siteURL := ""
	if config.ServiceSettings.SiteURL != nil {
		siteURL = *config.ServiceSettings.SiteURL
	}
	if !strings.HasSuffix(siteURL, "/") {
		siteURL += "/"
	}
	return siteURL + "plugins/" + pluginID
}

func getRelativeURL(config *model.Config) string {
	subpath, _ := utils.GetSubpathFromConfig(config)
	if !strings.HasSuffix(subpath, "/") {
		subpath += "/"
	}
	return subpath + "plugins/" + pluginID
}

func TestGetURL(t *testing.T) {
	testCases := []struct {
		Name     string
		URL      string
		Expected string
	}{
		{
			Name:     "no subpath, ending with /",
			URL:      "https://example.com/",
			Expected: "https://example.com/plugins/" + pluginID,
		},
		{
			Name:     "no subpath, not ending with /",
			URL:      "https://example.com",
			Expected: "https://example.com/plugins/" + pluginID,
		},
		{
			Name:     "with subpath, ending with /",
			URL:      "https://example.com/subpath/",
			Expected: "https://example.com/subpath/plugins/" + pluginID,
		},
		{
			Name:     "with subpath, not ending with /",
			URL:      "https://example.com/subpath",
			Expected: "https://example.com/subpath/plugins/" + pluginID,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			config := &model.Config{}
			config.SetDefaults()
			config.ServiceSettings.SiteURL = model.NewPointer(testCase.URL)

			actual := getURL(config)
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}

func TestGetRelativeURL(t *testing.T) {
	testCases := []struct {
		Name     string
		URL      string
		Expected string
	}{
		{
			Name:     "Empty URL",
			URL:      "",
			Expected: "/plugins/" + pluginID,
		},
		{
			Name:     "no subpath, ending with /",
			URL:      "https://example.com/",
			Expected: "/plugins/" + pluginID,
		},
		{
			Name:     "no subpath, not ending with /",
			URL:      "https://example.com",
			Expected: "/plugins/" + pluginID,
		},
		{
			Name:     "with subpath, ending with /",
			URL:      "https://example.com/subpath/",
			Expected: "/subpath/plugins/" + pluginID,
		},
		{
			Name:     "with subpath, not ending with /",
			URL:      "https://example.com/subpath",
			Expected: "/subpath/plugins/" + pluginID,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			config := &model.Config{}
			config.SetDefaults()
			config.ServiceSettings.SiteURL = model.NewPointer(testCase.URL)

			actual := getRelativeURL(config)
			assert.Equal(t, testCase.Expected, actual)
		})
	}
}
