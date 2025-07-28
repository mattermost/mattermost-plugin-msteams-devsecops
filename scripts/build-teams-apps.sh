#!/bin/bash

# Build Teams App Packages
# Main script for building Microsoft Teams app packages from the AppStore directory

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/teams-app-utils.sh"

# Default configuration
DEFAULT_APPSTORE_DIR="appstore"
DEFAULT_OUTPUT_DIR="dist/teams-apps"
DEFAULT_ACTION="build-all"

# Script configuration
APPSTORE_DIR="${APPSTORE_DIR:-$DEFAULT_APPSTORE_DIR}"
OUTPUT_DIR="${OUTPUT_DIR:-$DEFAULT_OUTPUT_DIR}"
ACTION="${ACTION:-$DEFAULT_ACTION}"
SPECIFIC_APP="${SPECIFIC_APP:-}"

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build Microsoft Teams app packages from the AppStore directory.

OPTIONS:
    -a, --action ACTION     Action to perform (build-all, build-one, list, clean)
    -s, --app APP_NAME      Specific app to build (for build-one action)
    -i, --input DIR         AppStore directory (default: $DEFAULT_APPSTORE_DIR)
    -o, --output DIR        Output directory (default: $DEFAULT_OUTPUT_DIR)
    -h, --help              Show this help message

ACTIONS:
    build-all               Build all Teams apps (default)
    build-one               Build a specific Teams app (requires --app)
    list                    List available apps and existing packages
    clean                   Clean up generated packages

EXAMPLES:
    $0                                          # Build all apps
    $0 --action build-one --app Corpus         # Build only Corpus app
    $0 --action list                            # List apps and packages
    $0 --action clean                           # Clean up packages
    $0 --output dist/my-apps                    # Use custom output directory

ENVIRONMENT VARIABLES:
    APPSTORE_DIR            Override default AppStore directory
    OUTPUT_DIR              Override default output directory
    ACTION                  Override default action
    SPECIFIC_APP            Override specific app name

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -a|--action)
                ACTION="$2"
                shift 2
                ;;
            -s|--app)
                SPECIFIC_APP="$2"
                shift 2
                ;;
            -i|--input)
                APPSTORE_DIR="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Validate arguments
validate_args() {
    # Check if AppStore directory exists
    if [ ! -d "$APPSTORE_DIR" ]; then
        log_error "AppStore directory not found: $APPSTORE_DIR"
        exit 1
    fi

    # Validate action
    case "$ACTION" in
        build-all|build-one|list|clean)
            ;;
        *)
            log_error "Invalid action: $ACTION"
            log_error "Valid actions: build-all, build-one, list, clean"
            exit 1
            ;;
    esac

    # Validate specific app if building one
    if [ "$ACTION" == "build-one" ] && [ -z "$SPECIFIC_APP" ]; then
        log_error "Specific app name required for build-one action"
        log_error "Use --app APP_NAME or set SPECIFIC_APP environment variable"
        exit 1
    fi

    # Check if specific app exists
    if [ -n "$SPECIFIC_APP" ]; then
        if [ ! -d "$APPSTORE_DIR/$SPECIFIC_APP" ]; then
            log_error "App not found: $APPSTORE_DIR/$SPECIFIC_APP"
            exit 1
        fi
    fi
}

# Build all Teams apps
build_all_apps() {
    log_info "Building all Teams apps from $APPSTORE_DIR"

    local apps
    apps=$(get_all_teams_apps "$APPSTORE_DIR")

    if [ -z "$apps" ]; then
        log_warning "No Teams apps found in $APPSTORE_DIR"
        return 0
    fi

    local total_apps=0
    local successful_builds=0
    local failed_builds=0

    while IFS= read -r app; do
        if [ -z "$app" ]; then
            continue
        fi

        total_apps=$((total_apps + 1))
        log_info "Processing app: $app"

        local app_dir="$APPSTORE_DIR/$app"

        # Validate app
        if validate_teams_app "$app_dir" "$app"; then
            # Build package
            if create_app_package "$app_dir" "$app" "$OUTPUT_DIR"; then
                successful_builds=$((successful_builds + 1))
            else
                failed_builds=$((failed_builds + 1))
            fi
        else
            log_error "Skipping invalid app: $app"
            failed_builds=$((failed_builds + 1))
        fi

        echo # Empty line for readability
    done <<< "$apps"

    # Summary
    log_info "Build Summary:"
    log_info "  Total apps: $total_apps"
    log_success "  Successful builds: $successful_builds"
    if [ $failed_builds -gt 0 ]; then
        log_error "  Failed builds: $failed_builds"
    fi

    if [ $successful_builds -gt 0 ]; then
        echo
        list_packages "$OUTPUT_DIR"
    fi

    return $([ $failed_builds -eq 0 ])
}

# Build a specific Teams app
build_specific_app() {
    local app="$SPECIFIC_APP"
    log_info "Building specific Teams app: $app"

    local app_dir="$APPSTORE_DIR/$app"

    # Validate app
    if ! validate_teams_app "$app_dir" "$app"; then
        log_error "App validation failed: $app"
        return 1
    fi

    # Build package
    if create_app_package "$app_dir" "$app" "$OUTPUT_DIR"; then
        log_success "Successfully built app: $app"
        echo
        list_packages "$OUTPUT_DIR"
        return 0
    else
        log_error "Failed to build app: $app"
        return 1
    fi
}

# List available apps and existing packages
list_apps_and_packages() {
    log_info "Available Teams apps in $APPSTORE_DIR:"

    local apps
    apps=$(get_all_teams_apps "$APPSTORE_DIR")

    if [ -z "$apps" ]; then
        log_warning "No Teams apps found in $APPSTORE_DIR"
    else
        while IFS= read -r app; do
            if [ -z "$app" ]; then
                continue
            fi

            local app_dir="$APPSTORE_DIR/$app"
            local manifest_file="$app_dir/manifest.json"

            if [ -f "$manifest_file" ]; then
                local version
                version=$(get_app_version "$manifest_file" 2>/dev/null) || version="unknown"
                printf "  %-30s (v%s)\n" "$app" "$version"
            else
                printf "  %-30s (no manifest)\n" "$app"
            fi
        done <<< "$apps"
    fi

    echo
    list_packages "$OUTPUT_DIR"
}

# Clean up generated packages
clean_packages_action() {
    log_info "Cleaning up Teams app packages"
    clean_packages "$OUTPUT_DIR"
}

# Main function
main() {
    # Parse arguments
    parse_args "$@"

    # Show configuration
    log_info "Teams App Builder Configuration:"
    log_info "  AppStore Directory: $APPSTORE_DIR"
    log_info "  Output Directory: $OUTPUT_DIR"
    log_info "  Action: $ACTION"
    if [ -n "$SPECIFIC_APP" ]; then
        log_info "  Specific App: $SPECIFIC_APP"
    fi
    echo

    # Validate arguments
    validate_args

    # Check dependencies
    if ! check_dependencies; then
        exit 1
    fi

    # Execute action
    case "$ACTION" in
        build-all)
            build_all_apps
            ;;
        build-one)
            build_specific_app
            ;;
        list)
            list_apps_and_packages
            ;;
        clean)
            clean_packages_action
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
