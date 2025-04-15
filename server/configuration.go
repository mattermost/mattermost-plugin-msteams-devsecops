package main

import (
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
	AppVersion                       string `json:"appVersion"`
	AppID                            string `json:"appID"`
	AppClientID                      string `json:"appClientID"`
	AppClientSecret                  string `json:"appClientSecret"`
	AppName                          string `json:"appName"`
	TenantID                         string `json:"tenantID"`
	DisableCheckCredentials          bool   `json:"internalDisableCheckCredentials"`
	DisableUserActivityNotifications bool   `json:"disableUserActivityNotifications"`
}

func (c *configuration) ProcessConfiguration() {
	c.TenantID = strings.TrimSpace(c.TenantID)
	c.AppClientID = strings.TrimSpace(c.AppClientID)
	c.AppClientSecret = strings.TrimSpace(c.AppClientSecret)
}

func (p *Plugin) validateConfiguration(configuration *configuration) error {
	configuration.ProcessConfiguration()

	if configuration.AppVersion == "" {
		return errors.New("application version should not be empty")
	}
	if configuration.AppID == "" {
		return errors.New("application ID should not be empty")
	}
	if configuration.TenantID == "" {
		return errors.New("tenant ID should not be empty")
	}
	if configuration.AppClientID == "" {
		return errors.New("client ID should not be empty")
	}
	if configuration.AppClientSecret == "" {
		return errors.New("client secret should not be empty")
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

	// Log before loading config
	p.API.LogDebug("Before LoadPluginConfiguration", "config_pointer", newConfig)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(newConfig); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	// Log after loading config with all field values
	p.API.LogDebug("Loaded plugin configuration",
		"AppVersion", newConfig.AppVersion,
		"AppID", newConfig.AppID,
		"TenantID", newConfig.TenantID,
		"AppClientID", newConfig.AppClientID,
		"AppClientSecret_empty", newConfig.AppClientSecret == "",
		"AppName", newConfig.AppName,
	)

	// Get the current configuration
	currentConfig := p.getConfiguration()
	if currentConfig != nil {
		// Handle JSON deserialization issue by implementing a custom merge strategy
		// If we have a field in the current config but it's missing in the new config,
		// it's likely due to JSON deserialization issues in Mattermost's plugin API
		if currentConfig.AppVersion != "" && newConfig.AppVersion == "" {
			newConfig.AppVersion = currentConfig.AppVersion
			p.API.LogDebug("Fixed missing AppVersion field", "AppVersion", newConfig.AppVersion)
		}
		if currentConfig.AppID != "" && newConfig.AppID == "" {
			newConfig.AppID = currentConfig.AppID
			p.API.LogDebug("Fixed missing AppID field", "AppID", newConfig.AppID)
		}
		if currentConfig.TenantID != "" && newConfig.TenantID == "" {
			newConfig.TenantID = currentConfig.TenantID
			p.API.LogDebug("Fixed missing TenantID field", "TenantID", newConfig.TenantID)
		}
		if currentConfig.AppClientID != "" && newConfig.AppClientID == "" {
			newConfig.AppClientID = currentConfig.AppClientID
			p.API.LogDebug("Fixed missing AppClientID field", "AppClientID", newConfig.AppClientID)
		}
		if currentConfig.AppClientSecret != "" && newConfig.AppClientSecret == "" {
			newConfig.AppClientSecret = currentConfig.AppClientSecret
			p.API.LogDebug("Fixed missing AppClientSecret field")
		}
		if currentConfig.AppName != "" && newConfig.AppName == "" {
			newConfig.AppName = currentConfig.AppName
			p.API.LogDebug("Fixed missing AppName field", "AppName", newConfig.AppName)
		}
	}

	// Save a copy of the configuration before setting it
	configCopy := newConfig.Clone()

	// Apply the new configuration
	p.setConfiguration(newConfig)

	// Log after setting config to verify no changes occurred
	p.API.LogDebug("After setConfiguration",
		"Original_AppVersion", configCopy.AppVersion,
		"Current_AppVersion", newConfig.AppVersion,
		"Original_AppID", configCopy.AppID,
		"Current_AppID", newConfig.AppID,
		"Original_TenantID", configCopy.TenantID,
		"Current_TenantID", newConfig.TenantID,
	)

	// Process and validate the configuration
	if err := p.validateConfiguration(newConfig); err != nil {
		return err
	}

	// Only restart the application if the OnActivate is already executed
	if p.pluginStore != nil {
		// Using a synchronous restart instead of async to avoid potential race conditions
		p.restart()
	}

	return nil
}
