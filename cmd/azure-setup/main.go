// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go-core/authentication"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "azure-setup",
	Short: "Automate Azure AD setup for Mattermost MS Teams plugin",
	Long: `Azure Setup Tool for Mattermost Mission Collaboration for Microsoft

This tool automates the Azure AD application registration and configuration
process required for the Mattermost MS Teams plugin. It handles:

  • App registration creation/update
  • API permissions configuration
  • Application ID URI and scope setup
  • Pre-authorized client applications
  • Client secret generation

The tool requires an authenticated Azure account with permissions to manage
applications in your Azure AD tenant.`,
	Version: version,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update Azure AD application registration",
	Long: `Create a new Azure AD application or update an existing one with the
required configuration for the Mattermost MS Teams plugin.

This command will:
  1. Authenticate to Azure
  2. Validate your permissions
  3. Create or update the application
  4. Configure API permissions (User.Read, TeamsActivity.Send, AppCatalog.Read.All)
  5. Set up API exposure with access_as_user scope
  6. Add pre-authorized Microsoft clients (Teams, Outlook)
  7. Generate a client secret

Example:
  azure-setup create --site-url https://mattermost.example.com --app-name "Mattermost for Teams"
  azure-setup create --site-url https://mm.example.com --client-id abc123... --dry-run`,
	RunE: runCreate,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Azure credentials and permissions",
	Long: `Validate that you can authenticate to Azure and have the necessary
permissions to create and manage applications.

This command performs a dry-run check without making any changes.`,
	RunE: runValidate,
}

// Command flags
var (
	flagTenantID         string
	flagSiteURL          string
	flagAppName          string
	flagClientID         string
	flagSecretExpiration int
	flagDryRun           bool
	flagNonInteractive   bool
	flagVerbose          bool
	flagOutputFormat     string
)

func init() {
	// Create command flags
	createCmd.Flags().StringVar(&flagTenantID, "tenant-id", "", "Azure AD Tenant ID (optional)")
	createCmd.Flags().StringVar(&flagSiteURL, "site-url", "", "Mattermost site URL (required)")
	createCmd.Flags().StringVar(&flagAppName, "app-name", "Mattermost for Teams", "Application display name")
	createCmd.Flags().StringVar(&flagClientID, "client-id", "", "Existing application client ID (for updates)")
	createCmd.Flags().IntVar(&flagSecretExpiration, "secret-expiration", 12, "Client secret expiration in months (1-24)")
	createCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview changes without applying them")
	createCmd.Flags().BoolVar(&flagNonInteractive, "non-interactive", false, "Run in non-interactive mode")
	createCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
	createCmd.Flags().StringVarP(&flagOutputFormat, "output", "o", "human", "Output format (human, json, env, mattermost)")

	createCmd.MarkFlagRequired("site-url")

	// Validate command flags
	validateCmd.Flags().StringVar(&flagTenantID, "tenant-id", "", "Azure AD Tenant ID (optional)")
	validateCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(validateCmd)
}

// runCreate executes the create command
func runCreate(cmd *cobra.Command, args []string) error {
	// Set a reasonable timeout for the entire operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build configuration
	config := &SetupConfig{
		TenantID:          flagTenantID,
		MattermostSiteURL: flagSiteURL,
		AppName:           flagAppName,
		ClientID:          flagClientID,
		SecretExpiration:  flagSecretExpiration,
		DryRun:            flagDryRun,
		NonInteractive:    flagNonInteractive,
		Verbose:           flagVerbose,
		OutputFormat:      flagOutputFormat,
		ctx:               ctx,
		rollback:          []func() error{},
	}

	// Validate inputs
	if err := validateInputs(config); err != nil {
		return errors.Wrap(err, "invalid input")
	}

	// Authenticate to Azure
	cred, err := authenticateToAzure(ctx, config.TenantID, config.Verbose)
	if err != nil {
		return errors.Wrap(err, "authentication failed")
	}
	config.credential = cred

	// Validate Azure connection
	if err := validateAzureConnection(ctx, cred, config.Verbose); err != nil {
		return errors.Wrap(err, "Azure connection validation failed")
	}

	// Create Graph client
	authProvider, err := authentication.NewAzureIdentityAuthenticationProvider(cred)
	if err != nil {
		return errors.Wrap(err, "failed to create auth provider")
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return errors.Wrap(err, "failed to create Graph adapter")
	}

	client := msgraphsdk.NewGraphServiceClient(adapter)

	// Validate permissions
	if err := validatePermissions(ctx, client, config.Verbose); err != nil {
		return errors.Wrap(err, "permission validation failed")
	}

	// Check for existing app
	existingApp, err := checkExistingApp(ctx, client, config.AppName, config.ClientID, config.Verbose)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing application")
	}

	// Execute setup with rollback on error
	result, err := executeSetup(ctx, client, config, existingApp)
	if err != nil {
		if !config.DryRun {
			executeRollback(config)
		}
		return err
	}

	// Output results
	return OutputResult(result, config.OutputFormat)
}

// runValidate executes the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	// Set a reasonable timeout for validation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("🔍 Validating Azure credentials and permissions...")

	// Authenticate to Azure
	cred, err := authenticateToAzure(ctx, flagTenantID, flagVerbose)
	if err != nil {
		return errors.Wrap(err, "authentication failed")
	}

	// Validate Azure connection
	if err := validateAzureConnection(ctx, cred, flagVerbose); err != nil {
		return errors.Wrap(err, "Azure connection validation failed")
	}

	// Create Graph client
	authProvider, err := authentication.NewAzureIdentityAuthenticationProvider(cred)
	if err != nil {
		return errors.Wrap(err, "failed to create auth provider")
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return errors.Wrap(err, "failed to create Graph adapter")
	}

	client := msgraphsdk.NewGraphServiceClient(adapter)

	// Validate permissions
	if err := validatePermissions(ctx, client, flagVerbose); err != nil {
		return errors.Wrap(err, "permission validation failed")
	}

	fmt.Println("\n✅ Validation complete - you are ready to create applications")
	return nil
}

// executeSetup orchestrates the entire setup process
func executeSetup(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *SetupConfig, existingApp models.Applicationable) (*SetupResult, error) {
	var app models.Applicationable
	var created bool
	var err error

	// Create or update application
	app, created, err = createOrUpdateApp(ctx, client, config, existingApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create/update application")
	}

	// Configure API permissions
	if err := configureAPIPermissions(ctx, client, config, app); err != nil {
		return nil, errors.Wrap(err, "failed to configure API permissions")
	}

	// Configure API exposure
	if err := configureAPIExposure(ctx, client, config, app); err != nil {
		return nil, errors.Wrap(err, "failed to configure API exposure")
	}

	// Generate client secret
	var clientSecret string
	var secretExpiration time.Time
	if !config.DryRun {
		clientSecret, secretExpiration, err = generateClientSecret(ctx, client, config, app)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate client secret")
		}
	} else {
		secretExpiration = time.Now().AddDate(0, config.SecretExpiration, 0)
		clientSecret = "***DRY-RUN-NO-SECRET-GENERATED***"
	}

	// Build application ID URI
	appIDURI, err := buildApplicationIDURI(config.MattermostSiteURL, *app.GetAppId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to build Application ID URI")
	}

	// Get tenant ID - either from config or from Azure
	tenantID := config.TenantID
	if tenantID == "" {
		// Try to get tenant ID from the organization
		tenantID, err = getTenantID(ctx, client, config.Verbose)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get tenant ID - please provide --tenant-id flag")
		}
	}

	// Build result
	result := &SetupResult{
		Success:             true,
		Message:             "Setup completed successfully",
		ApplicationClientID: *app.GetAppId(),
		TenantID:            tenantID,
		ClientSecret:        clientSecret,
		SecretExpiration:    secretExpiration.Format("2006-01-02 15:04:05 MST"),
		ApplicationID:       *app.GetId(),
		ApplicationName:     *app.GetDisplayName(),
		ApplicationIDURI:    appIDURI,
		Created:             created,
		DryRun:              config.DryRun,
	}

	return result, nil
}
