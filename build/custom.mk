# Include custom targets and environment variables here

APPSTORE_DIR ?= appstore
APPSTORE_DIST_DIR ?= dist/appstore

## Build zip bundles for each app under $(APPSTORE_DIR), named <folder>-<version>.zip
## where <version> is read from each folder's manifest.json `.version` field.
.PHONY: appstore-bundles
appstore-bundles:
	@command -v jq >/dev/null 2>&1 || { echo "appstore-bundles requires jq to be installed"; exit 1; }
	@mkdir -p "$(APPSTORE_DIST_DIR)"
	@find "$(APPSTORE_DIST_DIR)" -maxdepth 1 -type f -name '*.zip' -delete
	@find "$(APPSTORE_DIR)" -mindepth 1 -maxdepth 1 -type d | while IFS= read -r dir; do \
		name=$$(basename "$$dir"); \
		manifest="$$dir/manifest.json"; \
		if [ ! -f "$$manifest" ]; then \
			echo "Skipping $$name: missing manifest.json"; \
			continue; \
		fi; \
		version=$$(jq -r '.version // empty' "$$manifest"); \
		if [ -z "$$version" ]; then \
			echo "Skipping $$name: no .version in manifest.json"; \
			continue; \
		fi; \
		if ! printf '%s' "$$version" | grep -Eq '^[A-Za-z0-9._-]+$$'; then \
			echo "Skipping $$name: invalid version '$$version' for filename use"; \
			continue; \
		fi; \
		out="$$PWD/$(APPSTORE_DIST_DIR)/$$name-$$version.zip"; \
		echo "Bundling $$name-$$version.zip"; \
		(cd "$$dir" && zip -qr "$$out" . -x "*.zip") || exit $$?; \
	done
