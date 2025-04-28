// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func TestUpdateFrameAncestors(t *testing.T) {
	th := setupTestHelper(t)

	tests := []struct {
		name             string
		initialAncestors *string
		expectedDomains  []string
		shouldContainAll bool
	}{
		{
			name:             "Empty initial ancestors",
			initialAncestors: nil,
			expectedDomains: []string{
				"*.cloud.microsoft", "*.microsoft365.com", "*.office.com",
				"*.teams.microsoft.com", "outlook-sdf.office.com",
				"outlook-sdf.office365.com", "outlook.office.com",
				"outlook.office365.com", "teams.microsoft.com",
			},
			shouldContainAll: true,
		},
		{
			name:             "With existing ancestors",
			initialAncestors: stringPtr("example.com test.com"),
			expectedDomains: []string{
				"example.com", "test.com", "*.cloud.microsoft",
				"*.microsoft365.com", "*.office.com", "*.teams.microsoft.com",
			},
			shouldContainAll: true,
		},
		{
			name:             "With duplicate ancestors",
			initialAncestors: stringPtr("teams.microsoft.com example.com"),
			expectedDomains:  []string{"teams.microsoft.com", "example.com"},
			shouldContainAll: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the current config
			config := th.p.client.Configuration.GetConfig()
			require.NotNil(t, config)

			// Set the initial frame ancestors
			config.ServiceSettings.FrameAncestors = tt.initialAncestors
			err := th.p.client.Configuration.SaveConfig(config)
			require.NoError(t, err)

			// Call the function
			err = th.p.updateFrameAncestors()
			require.NoError(t, err)

			// Get the updated config
			updatedConfig := th.p.client.Configuration.GetConfig()
			require.NotNil(t, updatedConfig)
			require.NotNil(t, updatedConfig.ServiceSettings.FrameAncestors)

			// Check that the frame ancestors contain all expected domains
			frameAncestors := *updatedConfig.ServiceSettings.FrameAncestors
			ancestorsList := strings.Fields(frameAncestors)

			if tt.shouldContainAll {
				for _, domain := range tt.expectedDomains {
					assert.Contains(t, ancestorsList, domain)
				}
			}

			// Check that the list is sorted
			sortedList := make([]string, len(ancestorsList))
			copy(sortedList, ancestorsList)
			sort.Strings(sortedList)
			assert.Equal(t, sortedList, ancestorsList, "Frame ancestors should be sorted alphabetically")

			// Verify no duplicates
			uniqueDomains := make(map[string]bool)
			for _, domain := range ancestorsList {
				uniqueDomains[domain] = true
			}
			assert.Equal(t, len(uniqueDomains), len(ancestorsList), "Frame ancestors should have no duplicates")
		})
	}
}
