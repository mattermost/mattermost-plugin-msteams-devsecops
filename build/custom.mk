# Include custom targets and environment variables here

APPSTORE_DIR ?= appstore
APPSTORE_DIST_DIR ?= dist/appstore

## Resize source assets into submission-ready icons for each app under $(APPSTORE_DIR).
## Reads assets/logo-color.png (→ logo-color.png, 192×192) and
## assets/logo-outline.png (→ logo-outline.png, 32×32) in each app folder.
## Generated files are committed to git.
## Requires sips (macOS) or ImageMagick (magick/convert).
.PHONY: appstore-icons
appstore-icons:
	@command -v sips >/dev/null 2>&1 || command -v magick >/dev/null 2>&1 || command -v convert >/dev/null 2>&1 || \
		{ echo "appstore-icons requires sips (macOS) or ImageMagick (magick/convert)"; exit 1; }
	@find "$(APPSTORE_DIR)" -mindepth 1 -maxdepth 1 -type d | while IFS= read -r dir; do \
		name=$$(basename "$$dir"); \
		color_src="$$dir/assets/logo-color.png"; \
		outline_src="$$dir/assets/logo-outline.png"; \
		if [ ! -f "$$color_src" ] || [ ! -f "$$outline_src" ]; then \
			echo "Skipping $$name: missing assets/logo-color.png or assets/logo-outline.png"; continue; \
		fi; \
		echo "Generating icons for $$name"; \
		if command -v sips >/dev/null 2>&1; then \
			sips -z 192 192 "$$color_src" --out "$$dir/logo-color.png" >/dev/null || exit 1; \
			sips -z 32 32 "$$outline_src" --out "$$dir/logo-outline.png" >/dev/null || exit 1; \
		elif command -v magick >/dev/null 2>&1; then \
			magick "$$color_src" -resize 192x192! "$$dir/logo-color.png" || exit 1; \
			magick "$$outline_src" -resize 32x32! "$$dir/logo-outline.png" || exit 1; \
		else \
			convert "$$color_src" -resize 192x192! "$$dir/logo-color.png" || exit 1; \
			convert "$$outline_src" -resize 32x32! "$$dir/logo-outline.png" || exit 1; \
		fi; \
	done

## Build zip bundles for each app under $(APPSTORE_DIR), named <folder>-<version>.zip
## where <version> is read from each folder's manifest.json `.version` field.
## The assets/ and marketplace/ folders are not included in bundles.
## Requires jq; icons are generated via the appstore-icons dependency.
.PHONY: appstore-bundles
appstore-bundles: appstore-icons
	@command -v jq >/dev/null 2>&1 || { echo "appstore-bundles requires jq to be installed"; exit 1; }
	@mkdir -p "$(APPSTORE_DIST_DIR)"
	@find "$(APPSTORE_DIST_DIR)" -maxdepth 1 -type f -name '*.zip' -delete
	@find "$(APPSTORE_DIR)" -mindepth 1 -maxdepth 1 -type d | while IFS= read -r dir; do \
		name=$$(basename "$$dir"); \
		manifest="$$dir/manifest.json"; \
		if [ ! -f "$$manifest" ]; then echo "Skipping $$name: missing manifest.json"; continue; fi; \
		version=$$(jq -r '.version // empty' "$$manifest"); \
		if [ -z "$$version" ]; then echo "Skipping $$name: no .version in manifest.json"; continue; fi; \
		if ! printf '%s' "$$version" | grep -Eq '^[A-Za-z0-9._-]+$$'; then \
			echo "Skipping $$name: invalid version '$$version' for filename use"; continue; \
		fi; \
		out="$$PWD/$(APPSTORE_DIST_DIR)/$$name-$$version.zip"; \
		echo "Bundling $$name-$$version.zip"; \
		(cd "$$dir" && zip -qr "$$out" . -x "assets/*" -x "marketplace/*") || exit 1; \
	done
