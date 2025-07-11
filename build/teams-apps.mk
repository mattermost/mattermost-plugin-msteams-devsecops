# Teams App Builder Makefile
# Targets for building Microsoft Teams app packages

# Configuration
TEAMS_APPS_SCRIPT ?= scripts/build-teams-apps.sh
TEAMS_APPSTORE_DIR ?= appstore
TEAMS_OUTPUT_DIR ?= dist/teams-apps
TEAMS_APP ?=

# Check if the build script exists
ifeq ($(wildcard $(TEAMS_APPS_SCRIPT)),)
    $(error "Teams app build script not found: $(TEAMS_APPS_SCRIPT)")
endif

# Ensure the script is executable
$(shell chmod +x $(TEAMS_APPS_SCRIPT))

## Build all Teams app packages
.PHONY: build-teams-apps
build-teams-apps:
	@echo "Building all Teams app packages..."
	@APPSTORE_DIR=$(TEAMS_APPSTORE_DIR) \
	 OUTPUT_DIR=$(TEAMS_OUTPUT_DIR) \
	 ACTION=build-all \
	 $(TEAMS_APPS_SCRIPT)

## Build a specific Teams app package
.PHONY: build-teams-app
build-teams-app:
ifeq ($(TEAMS_APP),)
	@echo "Error: TEAMS_APP must be specified"
	@echo "Usage: make build-teams-app TEAMS_APP=<app_name>"
	@echo "Available apps:"
	@ls -1 $(TEAMS_APPSTORE_DIR)/ | grep -v '^\.' | sed 's/^/  /'
	@exit 1
else
	@echo "Building Teams app: $(TEAMS_APP)"
	@APPSTORE_DIR=$(TEAMS_APPSTORE_DIR) \
	 OUTPUT_DIR=$(TEAMS_OUTPUT_DIR) \
	 ACTION=build-one \
	 SPECIFIC_APP=$(TEAMS_APP) \
	 $(TEAMS_APPS_SCRIPT)
endif

## List available Teams apps and existing packages
.PHONY: list-teams-apps
list-teams-apps:
	@APPSTORE_DIR=$(TEAMS_APPSTORE_DIR) \
	 OUTPUT_DIR=$(TEAMS_OUTPUT_DIR) \
	 ACTION=list \
	 $(TEAMS_APPS_SCRIPT)

## Clean up generated Teams app packages
.PHONY: clean-teams-apps
clean-teams-apps:
	@echo "Cleaning up Teams app packages..."
	@APPSTORE_DIR=$(TEAMS_APPSTORE_DIR) \
	 OUTPUT_DIR=$(TEAMS_OUTPUT_DIR) \
	 ACTION=clean \
	 $(TEAMS_APPS_SCRIPT)

## Teams app help
.PHONY: help-teams-apps
help-teams-apps:
	@echo "Teams App Builder - Available Targets:"
	@echo ""
	@echo "  build-teams-apps           Build all Teams app packages"
	@echo "  build-teams-app            Build a specific Teams app package"
	@echo "                             Usage: make build-teams-app TEAMS_APP=<app_name>"
	@echo "  list-teams-apps            List available apps and existing packages"
	@echo "  clean-teams-apps           Clean up generated packages"
	@echo "  help-teams-apps            Show this help message"
	@echo ""
	@echo "Configuration Variables:"
	@echo "  TEAMS_APPSTORE_DIR         AppStore directory (default: $(TEAMS_APPSTORE_DIR))"
	@echo "  TEAMS_OUTPUT_DIR           Output directory (default: $(TEAMS_OUTPUT_DIR))"
	@echo "  TEAMS_APP                  Specific app name for build-teams-app target"
	@echo ""
	@echo "Examples:"
	@echo "  make build-teams-apps                              # Build all apps"
	@echo "  make build-teams-app TEAMS_APP=Corpus             # Build only Corpus app"
	@echo "  make build-teams-app TEAMS_APP=\"Community for Mattermost\"  # Build app with spaces"
	@echo "  make list-teams-apps                               # List apps and packages"
	@echo "  make clean-teams-apps                              # Clean up packages"
	@echo ""
	@echo "For more detailed options, run the script directly:"
	@echo "  ./$(TEAMS_APPS_SCRIPT) --help"

# Add teams-apps to clean target dependencies
clean-teams-apps-deps: clean-teams-apps

# Integration with existing build system
# This allows 'make clean' to also clean Teams app packages
ifneq ($(wildcard Makefile),)
    # We're being included in the main Makefile
    clean: clean-teams-apps-deps
endif