// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pluginstore

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost/server/public/plugin"
)

type User struct {
	MattermostUserID string
	TeamsObjectID    string
	TeamsSSOUsername string
}

func NewUser(mattermostUserID, teamsObjectID, teamsSSOUsername string) *User {
	return &User{
		MattermostUserID: mattermostUserID,
		TeamsObjectID:    teamsObjectID,
		TeamsSSOUsername: teamsSSOUsername,
	}
}

type Store interface {
	StoreUser(user *User) error
	GetUser(mattermostUserID string) (*User, error)
	StoreAppID(tenantID, appID string) error
	GetAppID(tenantID string) (string, error)
	UserExists(mattermostUserID string) (bool, error)
	StoreIcon(iconType string, data []byte) error
	GetIcon(iconType string) ([]byte, error)
	DeleteIcon(iconType string) error
}

type PluginStore struct {
	API plugin.API
}

func NewPluginStore(api plugin.API) *PluginStore {
	return &PluginStore{API: api}
}

func (s *PluginStore) StoreUser(user *User) error {
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}

	appErr := s.API.KVSet(getUserKey(user.MattermostUserID), value)
	if appErr != nil {
		return fmt.Errorf("failed to store user: %w", appErr)
	}

	return nil
}

func (s *PluginStore) GetUser(mattermostUserID string) (*User, error) {
	userBytes, appErr := s.API.KVGet(getUserKey(mattermostUserID))
	if appErr != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", mattermostUserID, appErr)
	}

	if len(userBytes) == 0 {
		return nil, NewErrNotFound(fmt.Sprintf("user %s not found", mattermostUserID))
	}

	var user User
	err := json.Unmarshal(userBytes, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user %s: %w", mattermostUserID, err)
	}
	return &user, nil
}

// UserExists checks if a user exists in the plugin store.
func (s *PluginStore) UserExists(mattermostUserID string) (bool, error) {
	userBytes, appErr := s.API.KVGet(getUserKey(mattermostUserID))
	if appErr != nil {
		return false, fmt.Errorf("failed to check user existence %s: %w", mattermostUserID, appErr)
	}

	if len(userBytes) == 0 {
		return false, nil
	}

	return true, nil
}

func (s *PluginStore) StoreAppID(tenantID, appID string) error {
	appErr := s.API.KVSet(getAppIDKey(tenantID), []byte(appID))
	if appErr != nil {
		return fmt.Errorf("failed to store app ID: %w", appErr)
	}

	return nil
}

func (s *PluginStore) GetAppID(tenantID string) (string, error) {
	appIDBytes, appErr := s.API.KVGet(getAppIDKey(tenantID))
	if appErr != nil {
		return "", fmt.Errorf("failed to get app ID: %w", appErr)
	}

	if appIDBytes == nil {
		return "", NewErrNotFound("app ID not found")
	}

	return string(appIDBytes), nil
}

func getUserKey(mattermostUserID string) string {
	return fmt.Sprintf("user:%s", mattermostUserID)
}

func getAppIDKey(tenantID string) string {
	return "appid_" + tenantID
}

// StoreIcon stores icon data in the KV store
func (s *PluginStore) StoreIcon(iconType string, data []byte) error {
	appErr := s.API.KVSet(getIconKey(iconType), data)
	if appErr != nil {
		return fmt.Errorf("failed to store icon %s: %w", iconType, appErr)
	}
	return nil
}

// GetIcon retrieves icon data from the KV store
func (s *PluginStore) GetIcon(iconType string) ([]byte, error) {
	iconData, appErr := s.API.KVGet(getIconKey(iconType))
	if appErr != nil {
		return nil, fmt.Errorf("failed to get icon %s: %w", iconType, appErr)
	}

	if len(iconData) == 0 {
		return nil, NewErrNotFound(fmt.Sprintf("icon %s not found", iconType))
	}

	return iconData, nil
}

// DeleteIcon removes icon data from the KV store
func (s *PluginStore) DeleteIcon(iconType string) error {
	appErr := s.API.KVDelete(getIconKey(iconType))
	if appErr != nil {
		return fmt.Errorf("failed to delete icon %s: %w", iconType, appErr)
	}
	return nil
}

func getIconKey(iconType string) string {
	return fmt.Sprintf("icon:%s", iconType)
}
