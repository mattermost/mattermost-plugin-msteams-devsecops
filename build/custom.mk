# Include custom targets and environment variables here

APPSTORE_DIR ?= appstore
APPSTORE_DIST_DIR ?= dist/appstore

## Build zip bundles for each app under $(APPSTORE_DIR), named <folder>-<version>.zip
## where <version> is read from each folder's manifest.json `.version` field.
.PHONY: appstore-bundles
appstore-bundles:
	@mkdir -p $(APPSTORE_DIST_DIR)
	@find $(APPSTORE_DIR) -mindepth 1 -maxdepth 1 -type d | while IFS= read -r dir; do \
		name=$$(basename "$$dir"); \
		manifest="$$dir/manifest.json"; \
		if [ ! -f "$$manifest" ]; then \
			echo "Skipping $$name: missing manifest.json"; \
			continue; \
		fi; \
		version=$$(sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$$manifest" | head -1); \
		if [ -z "$$version" ]; then \
			echo "Skipping $$name: no .version in manifest.json"; \
			continue; \
		fi; \
		out="$$PWD/$(APPSTORE_DIST_DIR)/$$name-$$version.zip"; \
		echo "Bundling $$name-$$version.zip"; \
		rm -f "$$out"; \
		(cd "$$dir" && zip -qr "$$out" . -x "*.zip") || exit $$?; \
	done
