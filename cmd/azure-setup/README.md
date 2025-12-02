# Azure Setup Tool for Mattermost MS Teams Plugin

This CLI tool automates the Azure AD application registration and configuration process required for the **Mattermost Mission Collaboration for Microsoft** plugin.

## Features

✅ **Automated Azure App Registration**
- Creates new Azure AD applications or updates existing ones
- Configures single-tenant authentication

✅ **API Permissions Configuration**
- `User.Read` (Delegated) - User authentication
- `TeamsActivity.Send` (Application) - Send notifications to Teams
- `AppCatalog.Read.All` (Application) - App catalog operations

✅ **API Exposure Setup**
- Application ID URI: `api://{hostname}/{client-id}`
- `access_as_user` scope for SSO
- Pre-authorized Microsoft clients (Teams Web/Desktop, Outlook Web/Desktop)

✅ **Client Secret Generation**
- Secure secret generation with configurable expiration (1-24 months)
- One-time display with security warnings

✅ **Multiple Authentication Methods**
- Environment variables (Service Principal)
- Azure CLI
- Interactive browser
- Device code flow

✅ **Safety Features**
- Dry-run mode to preview changes
- Rollback on errors
- Idempotent operations
- Comprehensive validation

## Prerequisites

- **Azure Account** with permissions to manage applications
- **Required Azure AD Roles** (one of):
  - Application Administrator
  - Cloud Application Administrator
  - Global Administrator
- **Go 1.23+** (for building from source)

## Installation

### Build from Source

```bash
cd /path/to/mattermost-plugin-msteams-devsecops
go build -o azure-setup ./cmd/azure-setup/
```

### Build via Makefile

```bash
make azure-setup
# Binaries available in bin/ for multiple platforms
```

### Install to PATH

```bash
go install ./cmd/azure-setup
```

### Running Tests

```bash
# Run all tests
go test ./cmd/azure-setup/

# Run tests with coverage
go test -cover ./cmd/azure-setup/

# See TESTING.md for detailed testing documentation
```

## Quick Start

### 1. Validate Your Azure Access

```bash
azure-setup validate --verbose
```

This checks that you can authenticate to Azure and have the necessary permissions.

### 2. Create Azure Application (Dry Run)

```bash
azure-setup create \
  --site-url https://mattermost.example.com \
  --app-name "Mattermost for Teams" \
  --dry-run \
  --verbose
```

### 3. Create Azure Application (Live)

```bash
azure-setup create \
  --site-url https://mattermost.example.com \
  --app-name "Mattermost for Teams" \
  --verbose
```

### 4. View Output in Different Formats

```bash
# Human-readable (default)
azure-setup create --site-url https://mm.example.com --app-name "My App"

# JSON format (for scripting)
azure-setup create --site-url https://mm.example.com --app-name "My App" -o json

# Environment variables
azure-setup create --site-url https://mm.example.com --app-name "My App" -o env

# Mattermost config.json format
azure-setup create --site-url https://mm.example.com --app-name "My App" -o mattermost
```

## Commands

### `azure-setup create`

Create or update an Azure AD application with all required configuration.

**Flags:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--site-url` | string | ✅ Yes | - | Mattermost site URL (must be HTTPS) |
| `--app-name` | string | No | "Mattermost for Teams" | Application display name |
| `--tenant-id` | string | No | - | Azure AD Tenant ID (auto-detected if omitted) |
| `--client-id` | string | No | - | Existing app client ID (for updates) |
| `--secret-expiration` | int | No | 12 | Secret expiration in months (1-24) |
| `--dry-run` | bool | No | false | Preview changes without applying |
| `--verbose` / `-v` | bool | No | false | Enable verbose output |
| `--output` / `-o` | string | No | "human" | Output format: human, json, env, mattermost |
| `--non-interactive` | bool | No | false | Run without prompts |

**Examples:**

```bash
# Basic usage
azure-setup create --site-url https://mattermost.example.com

# Update existing app
azure-setup create \
  --site-url https://mattermost.example.com \
  --client-id "abc123-def456-..." \
  --verbose

# Custom secret expiration
azure-setup create \
  --site-url https://mattermost.example.com \
  --secret-expiration 24

# Output as environment variables
azure-setup create \
  --site-url https://mattermost.example.com \
  --output env > .env
```

### `azure-setup validate`

Validate Azure credentials and permissions without making changes.

**Flags:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--tenant-id` | string | No | - | Azure AD Tenant ID |
| `--verbose` / `-v` | bool | No | false | Enable verbose output |

**Example:**

```bash
azure-setup validate --verbose
```

## Authentication Methods

The tool tries multiple authentication methods in order:

### 1. Environment Variables (Service Principal)

Set these environment variables:

```bash
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"
```

### 2. Azure CLI

If you're logged in via Azure CLI:

```bash
az login
azure-setup create --site-url https://mattermost.example.com
```

### 3. Interactive Browser

Opens a browser window for interactive authentication.

### 4. Device Code Flow

For headless environments, provides a device code to enter on another device.

## Output Formats

### Human-Readable (default)

Provides clear, formatted output with next steps:

```
======================================================================
✅ AZURE SETUP COMPLETE
======================================================================

🔐 CREDENTIALS FOR MATTERMOST CONFIGURATION
----------------------------------------------------------------------
Tenant ID:             abc123...
Application Client ID: def456...
Client Secret:         secret-value
Secret Expires:        2026-01-15 12:00:00 UTC
----------------------------------------------------------------------

📝 NEXT STEPS
1. Grant admin consent for API permissions
2. Configure the Mattermost plugin
3. Download and install the Teams app manifest
```

### JSON Format

Machine-readable JSON for scripting:

```bash
azure-setup create --site-url https://mm.example.com -o json | jq .
```

### Environment Variables Format

Ready-to-use environment variable exports:

```bash
azure-setup create --site-url https://mm.example.com -o env >> .env
source .env
```

### Mattermost Config Format

Snippet to merge into Mattermost `config.json`:

```bash
azure-setup create --site-url https://mm.example.com -o mattermost > plugin-config.json
```

## Troubleshooting

### Authentication Failures

**Problem:** "Failed to authenticate to Azure"

**Solutions:**
- Ensure you're logged into Azure CLI: `az login`
- Or set service principal environment variables
- Check network connectivity to Azure

### Permission Errors

**Problem:** "Failed to create application: insufficient privileges"

**Solutions:**
- Verify you have one of the required Azure AD roles
- Contact your Azure AD administrator to grant permissions
- Check the role assignments in Azure Portal → Azure AD → Roles and administrators

### Dry Run Shows Errors

**Problem:** Errors appear even in dry-run mode

**Solution:** Dry-run validation errors indicate issues with input parameters, not Azure operations. Fix the parameters and try again.

## Security Best Practices

### Client Secret Management

🔒 **IMPORTANT:**
- Client secrets are shown **only once** - save them immediately
- Store secrets securely (e.g., Azure Key Vault, HashiCorp Vault)
- Never commit secrets to version control
- Rotate secrets before expiration (set calendar reminders)

### Principle of Least Privilege

- Only grant the minimum required API permissions
- Use service principals for automation (not user accounts)
- Regularly audit application permissions

### Secret Rotation

Set up secret rotation before expiration:

```bash
# Generate new secret with 12 month expiration
azure-setup create \
  --site-url https://mattermost.example.com \
  --client-id "existing-client-id" \
  --secret-expiration 12
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Setup Azure AD App

on:
  workflow_dispatch:
    inputs:
      site_url:
        description: 'Mattermost Site URL'
        required: true

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Build Azure Setup Tool
        run: go build -o azure-setup ./cmd/azure-setup/

      - name: Configure Azure Application
        env:
          AZURE_TENANT_ID: ${{ secrets.AZURE_TENANT_ID }}
          AZURE_CLIENT_ID: ${{ secrets.AZURE_CLIENT_ID }}
          AZURE_CLIENT_SECRET: ${{ secrets.AZURE_CLIENT_SECRET }}
        run: |
          ./azure-setup create \
            --site-url ${{ github.event.inputs.site_url }} \
            --output json \
            --non-interactive \
            > azure-config.json

      - name: Upload Configuration
        uses: actions/upload-artifact@v3
        with:
          name: azure-configuration
          path: azure-config.json
```

## Next Steps After Running the Tool

### 1. Grant Admin Consent

Visit the Azure Portal URL provided in the output to grant admin consent for API permissions.

### 2. Configure Mattermost Plugin

In Mattermost:
1. Go to **System Console > Plugins > MS Teams DevSecOps**
2. Enter the credentials from the tool output:
   - Tenant ID
   - Application Client ID
   - Client Secret
3. Save the configuration

### 3. Generate and Upload Teams App Manifest

In the Mattermost plugin settings:
1. Download the Teams app manifest
2. Go to **Microsoft Teams Admin Center**
3. Navigate to **Teams apps > Manage apps > Upload**
4. Upload the manifest ZIP file

### 4. Test the Integration

1. Install the app in Microsoft Teams
2. Add the Mattermost tab to a team
3. Verify SSO authentication works
4. Test notifications from Mattermost to Teams

## Support and Contributing

For issues, questions, or contributions:
- **GitHub Issues**: https://github.com/mattermost/mattermost-plugin-msteams-devsecops/issues
- **Documentation**: https://docs.mattermost.com/integrations-guide/mattermost-mission-collaboration-for-m365.html
- **Community**: https://community.mattermost.com/

## License

Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
See LICENSE.txt for license information.
