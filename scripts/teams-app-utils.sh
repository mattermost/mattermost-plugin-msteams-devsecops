#!/bin/bash

# Teams App Build Utilities
# Common functions for building Teams app packages

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if required commands are available
check_dependencies() {
    local missing_deps=()

    if ! command -v jq &> /dev/null; then
        missing_deps+=(jq)
    fi

    if ! command -v zip &> /dev/null; then
        missing_deps+=(zip)
    fi

    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again."
        return 1
    fi

    return 0
}

# Get all available Teams apps
get_all_teams_apps() {
    local appstore_dir="$1"

    if [ ! -d "$appstore_dir" ]; then
        log_error "AppStore directory not found: $appstore_dir"
        return 1
    fi

    find "$appstore_dir" -maxdepth 1 -type d -not -path "$appstore_dir" -exec basename {} \; | sort
}

# Validate if a directory is a valid Teams app
validate_teams_app() {
    local app_dir="$1"
    local app_name="$2"

    if [ ! -d "$app_dir" ]; then
        log_error "App directory not found: $app_dir"
        return 1
    fi

    local manifest_file="$app_dir/manifest.json"
    if [ ! -f "$manifest_file" ]; then
        log_error "manifest.json not found in $app_dir"
        return 1
    fi

    # Validate JSON syntax
    if ! jq empty "$manifest_file" 2>/dev/null; then
        log_error "Invalid JSON in $manifest_file"
        return 1
    fi

    # Check required fields
    local version
    version=$(jq -r '.version' "$manifest_file" 2>/dev/null)
    if [ "$version" == "null" ] || [ -z "$version" ]; then
        log_error "Missing or invalid version in $manifest_file"
        return 1
    fi

    local app_id
    app_id=$(jq -r '.id' "$manifest_file" 2>/dev/null)
    if [ "$app_id" == "null" ] || [ -z "$app_id" ]; then
        log_error "Missing or invalid id in $manifest_file"
        return 1
    fi

    # Check for required icon files
    local color_icon outline_icon
    color_icon=$(jq -r '.icons.color' "$manifest_file" 2>/dev/null)
    outline_icon=$(jq -r '.icons.outline' "$manifest_file" 2>/dev/null)

    if [ "$color_icon" != "null" ] && [ -n "$color_icon" ]; then
        if [ ! -f "$app_dir/$color_icon" ]; then
            log_warning "Color icon file not found: $app_dir/$color_icon"
        fi
    fi

    if [ "$outline_icon" != "null" ] && [ -n "$outline_icon" ]; then
        if [ ! -f "$app_dir/$outline_icon" ]; then
            log_warning "Outline icon file not found: $app_dir/$outline_icon"
        fi
    fi

    log_info "App '$app_name' validated successfully (version: $version)"
    return 0
}

# Extract version from manifest.json
get_app_version() {
    local manifest_file="$1"

    if [ ! -f "$manifest_file" ]; then
        log_error "Manifest file not found: $manifest_file"
        return 1
    fi

    local version
    version=$(jq -r '.version' "$manifest_file" 2>/dev/null)
    if [ "$version" == "null" ] || [ -z "$version" ]; then
        log_error "Could not extract version from $manifest_file"
        return 1
    fi

    echo "$version"
}

# Sanitize app name for file naming
sanitize_app_name() {
    local app_name="$1"
    # Replace spaces and special characters with hyphens
    echo "$app_name" | sed 's/[^a-zA-Z0-9._-]/-/g' | sed 's/-\+/-/g' | sed 's/^-\|-$//g'
}

# Create zip package for a Teams app
create_app_package() {
    local app_dir="$1"
    local app_name="$2"
    local output_dir="$3"

    if [ ! -d "$app_dir" ]; then
        log_error "App directory not found: $app_dir"
        return 1
    fi

    local manifest_file="$app_dir/manifest.json"
    local version
    version=$(get_app_version "$manifest_file")
    if [ $? -ne 0 ]; then
        return 1
    fi

    # Create output directory if it doesn't exist
    mkdir -p "$output_dir"

    # Sanitize app name for filename
    local sanitized_name
    sanitized_name=$(sanitize_app_name "$app_name")

    # Create zip file with version in name
    local zip_name="${sanitized_name}-${version}.zip"
    local zip_path="$output_dir/$zip_name"

    log_info "Creating package: $zip_name"

    # Create the zip file
    local current_dir
    current_dir=$(pwd)

    cd "$app_dir"
    if zip -r "$current_dir/$zip_path" . -x "*.DS_Store" -x "Thumbs.db" -x "*~" -x "*.tmp" > /dev/null 2>&1; then
        cd "$current_dir"
        log_success "Created package: $zip_path"

        # Display package info
        local file_size
        file_size=$(du -h "$zip_path" | cut -f1)
        log_info "Package size: $file_size"

        return 0
    else
        cd "$current_dir"
        log_error "Failed to create package: $zip_path"
        return 1
    fi
}

# List all files in a directory with their sizes
list_packages() {
    local output_dir="$1"

    if [ ! -d "$output_dir" ]; then
        log_info "No packages found (output directory doesn't exist)"
        return 0
    fi

    local packages
    packages=$(find "$output_dir" -name "*.zip" -type f 2>/dev/null)

    if [ -z "$packages" ]; then
        log_info "No packages found in $output_dir"
        return 0
    fi

    log_info "Available packages:"
    echo "$packages" | while read -r package; do
        local basename_pkg size
        basename_pkg=$(basename "$package")
        size=$(du -h "$package" | cut -f1)
        printf "  %-40s %s\n" "$basename_pkg" "$size"
    done
}

# Clean up old packages
clean_packages() {
    local output_dir="$1"

    if [ ! -d "$output_dir" ]; then
        log_info "Nothing to clean (output directory doesn't exist)"
        return 0
    fi

    local packages
    packages=$(find "$output_dir" -name "*.zip" -type f 2>/dev/null)

    if [ -z "$packages" ]; then
        log_info "Nothing to clean (no packages found)"
        return 0
    fi

    log_info "Cleaning up packages..."
    echo "$packages" | while read -r package; do
        rm -f "$package"
        log_info "Removed: $(basename "$package")"
    done

    # Remove empty directory
    rmdir "$output_dir" 2>/dev/null || true

    log_success "Cleanup completed"
}
