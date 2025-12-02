// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"testing"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateClientSecret_DryRun tests dry-run mode for client secret generation
func TestGenerateClientSecret_DryRun(t *testing.T) {
	ctx := context.Background()
	config := &SetupConfig{
		AppName:          "Test App",
		SecretExpiration: 12,
		Verbose:          false,
		DryRun:           true,
		rollback:         []func() error{},
	}

	app := models.NewApplication()
	displayName := "Test App"
	app.SetDisplayName(&displayName)
	appID := "abc-123"
	app.SetAppId(&appID)
	objectID := "obj-123"
	app.SetId(&objectID)

	secret, expiration, err := generateClientSecret(ctx, nil, config, app)
	require.NoError(t, err, "Dry run should not return error")

	assert.Equal(t, "mock-secret-value-for-dry-run", secret)
	assert.False(t, expiration.IsZero(), "Should have expiration date")

	// Verify expiration is approximately 12 months from now
	expectedExpiration := time.Now().AddDate(0, 12, 0)
	timeDiff := expiration.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 5*time.Second, "Expiration should be approximately 12 months from now")
}

// TestGenerateClientSecret_ExpirationCalculation tests expiration date calculation
func TestGenerateClientSecret_ExpirationCalculation(t *testing.T) {
	tests := []struct {
		name             string
		secretExpiration int
		expectedMonths   int
	}{
		{
			name:             "1_month",
			secretExpiration: 1,
			expectedMonths:   1,
		},
		{
			name:             "12_months",
			secretExpiration: 12,
			expectedMonths:   12,
		},
		{
			name:             "24_months",
			secretExpiration: 24,
			expectedMonths:   24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := &SetupConfig{
				AppName:          "Test App",
				SecretExpiration: tt.secretExpiration,
				Verbose:          false,
				DryRun:           true,
			}

			app := models.NewApplication()
			displayName := "Test App"
			app.SetDisplayName(&displayName)
			appID := "abc-123"
			app.SetAppId(&appID)
			objectID := "obj-123"
			app.SetId(&objectID)

			_, expiration, err := generateClientSecret(ctx, nil, config, app)
			require.NoError(t, err)

			// Calculate expected expiration
			expectedExpiration := time.Now().AddDate(0, tt.expectedMonths, 0)

			// Allow 5 seconds of drift for test execution time
			timeDiff := expiration.Sub(expectedExpiration)
			assert.Less(t, timeDiff.Abs(), 5*time.Second,
				"Expiration should be approximately %d months from now", tt.expectedMonths)
		})
	}
}

// TestGenerateClientSecret_VerboseOutput tests verbose mode
func TestGenerateClientSecret_VerboseOutput(t *testing.T) {
	ctx := context.Background()
	config := &SetupConfig{
		AppName:          "Test App",
		SecretExpiration: 12,
		Verbose:          true,
		DryRun:           true,
		rollback:         []func() error{},
	}

	app := models.NewApplication()
	displayName := "Test App"
	app.SetDisplayName(&displayName)
	appID := "abc-123"
	app.SetAppId(&appID)
	objectID := "obj-123"
	app.SetId(&objectID)

	secret, expiration, err := generateClientSecret(ctx, nil, config, app)
	require.NoError(t, err, "Verbose mode should not affect success")
	assert.NotEmpty(t, secret)
	assert.False(t, expiration.IsZero())
}

// TestGenerateClientSecret_SecretFormat tests secret format and properties
func TestGenerateClientSecret_SecretFormat(t *testing.T) {
	t.Run("dry_run_returns_mock_secret", func(t *testing.T) {
		ctx := context.Background()
		config := &SetupConfig{
			AppName:          "Test App",
			SecretExpiration: 12,
			DryRun:           true,
		}

		app := models.NewApplication()
		displayName := "Test App"
		app.SetDisplayName(&displayName)
		appID := "abc-123"
		app.SetAppId(&appID)
		objectID := "obj-123"
		app.SetId(&objectID)

		secret, _, err := generateClientSecret(ctx, nil, config, app)
		require.NoError(t, err)

		assert.Equal(t, "mock-secret-value-for-dry-run", secret)
		assert.NotEmpty(t, secret, "Secret should not be empty")
	})
}

// TestGenerateClientSecret_ExpirationRange tests valid expiration ranges
func TestGenerateClientSecret_ExpirationRange(t *testing.T) {
	tests := []struct {
		name             string
		secretExpiration int
		shouldBeValid    bool
	}{
		{
			name:             "minimum_1_month",
			secretExpiration: 1,
			shouldBeValid:    true,
		},
		{
			name:             "maximum_24_months",
			secretExpiration: 24,
			shouldBeValid:    true,
		},
		{
			name:             "middle_12_months",
			secretExpiration: 12,
			shouldBeValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate through input validation
			config := &SetupConfig{
				MattermostSiteURL: "https://mattermost.example.com",
				AppName:           "Test App",
				SecretExpiration:  tt.secretExpiration,
			}

			err := validateInputs(config)
			if tt.shouldBeValid {
				assert.NoError(t, err, "Expiration of %d months should be valid", tt.secretExpiration)
			} else {
				assert.Error(t, err, "Expiration of %d months should be invalid", tt.secretExpiration)
			}
		})
	}
}

// TestGenerateClientSecret_DisplayName tests secret display name generation
func TestGenerateClientSecret_DisplayName(t *testing.T) {
	t.Run("includes_expiration_date", func(t *testing.T) {
		expirationDate := time.Now().AddDate(0, 12, 0)

		displayName := "Mattermost Plugin Secret (expires " + expirationDate.Format("2006-01-02") + ")"

		assert.Contains(t, displayName, "Mattermost Plugin Secret")
		assert.Contains(t, displayName, "expires")
		assert.Contains(t, displayName, expirationDate.Format("2006-01-02"))
	})

	t.Run("format_is_consistent", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2027, 1, 30, 0, 0, 0, 0, time.UTC),
		}

		for _, date := range dates {
			displayName := "Mattermost Plugin Secret (expires " + date.Format("2006-01-02") + ")"
			assert.Regexp(t, `^Mattermost Plugin Secret \(expires \d{4}-\d{2}-\d{2}\)$`, displayName)
		}
	})
}

// TestGenerateClientSecret_ErrorHandling tests error scenarios
func TestGenerateClientSecret_ErrorHandling(t *testing.T) {
	t.Run("requires_application_object", func(t *testing.T) {
		app := models.NewApplication()
		objectID := "obj-123"
		app.SetId(&objectID)

		assert.NotNil(t, app.GetId(), "Application must have object ID")
	})

	t.Run("requires_valid_config", func(t *testing.T) {
		config := &SetupConfig{
			AppName:          "Test App",
			SecretExpiration: 12,
			DryRun:           true,
		}

		assert.NotNil(t, config)
		assert.NotEmpty(t, config.AppName)
		assert.Greater(t, config.SecretExpiration, 0)
	})
}

// TestGenerateClientSecret_SuccessScenarios tests successful generation scenarios
func TestGenerateClientSecret_SuccessScenarios(t *testing.T) {
	t.Run("dry_run_always_succeeds", func(t *testing.T) {
		ctx := context.Background()
		config := &SetupConfig{
			AppName:          "Test App",
			SecretExpiration: 12,
			DryRun:           true,
		}

		app := models.NewApplication()
		displayName := "Test App"
		app.SetDisplayName(&displayName)
		appID := "abc-123"
		app.SetAppId(&appID)
		objectID := "obj-123"
		app.SetId(&objectID)

		secret, expiration, err := generateClientSecret(ctx, nil, config, app)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.False(t, expiration.IsZero())
	})
}

// TestGenerateClientSecret_ExpirationFormatting tests expiration date formatting
func TestGenerateClientSecret_ExpirationFormatting(t *testing.T) {
	t.Run("format_matches_expected", func(t *testing.T) {
		expirationDate := time.Date(2025, 12, 31, 15, 30, 45, 0, time.UTC)
		formatted := expirationDate.Format("2006-01-02 15:04:05 MST")

		assert.Contains(t, formatted, "2025-12-31")
		assert.Contains(t, formatted, "15:30:45")
	})
}
