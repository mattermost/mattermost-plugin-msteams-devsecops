package main

import (
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	// M365 Settings
	M365TenantID     string `json:"m365_tenant_id"`
	M365ClientID     string `json:"m365_client_id"`
	M365ClientSecret string `json:"m365_client_secret"`

	// Plugin Settings
	DisableUserActivityNotifications bool `json:"disable_user_activity_notifications"`
	DisableCheckCredentials          bool `json:"internal_disable_check_credentials"`

	// Manifest Settings
	AppVersion      string `json:"app_version"`
	AppID           string `json:"app_id"`
	AppName         string `json:"app_name"`
	IconColorPath   string `json:"icon_color_path"`
	IconOutlinePath string `json:"icon_outline_path"`

	// Legacy
	LegacyTenantID        string `json:"tenantID"`        // Deprecated: use M365TenantID instead
	LegacyAppClientID     string `json:"appClientID"`     // Deprecated: use M365ClientID instead
	LegacyAppClientSecret string `json:"appClientSecret"` // Deprecated: use M365ClientSecret instead
	LegacyAppVersion      string `json:"appVersion"`      // Deprecated: use AppVersion instead
	LegacyAppID           string `json:"appID"`           // Deprecated: use AppID instead
	LegacyAppName         string `json:"appName"`         // Deprecated: use AppName instead
}

func (c *configuration) ProcessConfiguration() bool {
	hasChange := false

	// handle legacy configuration fields
	if c.M365TenantID == "" && c.LegacyTenantID != "" {
		c.M365TenantID = c.LegacyTenantID
		hasChange = true
	}
	if c.M365ClientID == "" && c.LegacyAppClientID != "" {
		c.M365ClientID = c.LegacyAppClientID
		hasChange = true
	}
	if c.M365ClientSecret == "" && c.LegacyAppClientSecret != "" {
		c.M365ClientSecret = c.LegacyAppClientSecret
		hasChange = true
	}

	if c.AppVersion == "" && c.LegacyAppVersion != "" {
		c.AppVersion = c.LegacyAppVersion
		hasChange = true
	}

	if c.AppID == "" && c.LegacyAppID != "" {
		c.AppID = c.LegacyAppID
		hasChange = true
	}

	if c.AppName == "" && c.LegacyAppName != "" {
		c.AppName = c.LegacyAppName
		hasChange = true
	}

	// Trim whitespace from key fields
	length := len(c.M365TenantID)
	c.M365TenantID = strings.TrimSpace(c.M365TenantID)
	if length != len(c.M365TenantID) {
		hasChange = true
	}

	length = len(c.M365ClientID)
	c.M365ClientID = strings.TrimSpace(c.M365ClientID)
	if length != len(c.M365ClientID) {
		hasChange = true
	}

	length = len(c.M365ClientSecret)
	c.M365ClientSecret = strings.TrimSpace(c.M365ClientSecret)
	if length != len(c.M365ClientSecret) {
		hasChange = true
	}

	return hasChange
}

func (p *Plugin) validateConfiguration(configuration *configuration) error {
	configuration.ProcessConfiguration()

	if configuration.M365TenantID == "" {
		return errors.New("tenant ID should not be empty")
	}
	if configuration.M365ClientID == "" {
		return errors.New("client ID should not be empty")
	}
	if configuration.M365ClientSecret == "" {
		return errors.New("client secret should not be empty")
	}

	if configuration.AppVersion == "" {
		return errors.New("application version should not be empty")
	}
	if configuration.AppID == "" {
		return errors.New("application ID should not be empty")
	}
	if configuration.AppName == "" {
		return errors.New("app name should not be empty")
	}
	return nil
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// ToMap converts the configuration struct to a map using JSON tags as keys.
func (c *configuration) ToMap() map[string]any {
	// Convert the configuration struct to a map
	configMap := make(map[string]any)
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i).Interface()

		// Get the JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			// If no JSON tag, fall back to field name
			configMap[field.Name] = value
			continue
		}

		// Handle JSON tag `-` which indicates to skip this field
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "-" {
			continue
		}

		configMap[jsonName] = value
	}
	return configMap
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	// Create a new configuration to hold the new settings
	var newConfig = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(newConfig); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if newConfig.ProcessConfiguration() {
		// If the configuration has changed, we need to save it back to the config store
		if err := p.API.SavePluginConfig(newConfig.ToMap()); err != nil {
			return errors.Wrap(err, "failed to set plugin configuration")
		}
	}

	// Validate the configuration
	/*
		Skip validation for now, as it is preventing the plugin from being installed
		since the configuration is not set yet.  OnConfigurationChange is called before
		OnActivate.

		if err := p.validateConfiguration(newConfig); err != nil {
			return err
		}
	*/

	// Apply the new configuration
	p.setConfiguration(newConfig)

	// Only restart the application if the OnActivate is already executed
	if p.pluginStore != nil {
		go p.restart()
	}

	return nil
}

// shouldSkipTokenValidation returns true if the token validation should be skipped based on
// the MM_DEVSECOPS_SKIP_TOKEN_VALIDATION environment var is set to true.
func shouldSkipTokenValidation() bool {
	if skipTokenValidation, ok := os.LookupEnv("MM_DEVSECOPS_SKIP_TOKEN_VALIDATION"); ok {
		switch strings.ToLower(skipTokenValidation) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}
