---
model: claude-sonnet-4-6
---

# Microsoft Teams App Store Submission Reviewer

Review every app package under `appstore/` in this repository and report whether each one
complies with Microsoft's current Teams Store submission requirements.

**Do not rely on training-data knowledge for any guideline values. Always fetch the official
documentation first and derive all rules from what you read there.**

## Step 1 — Fetch the official guidelines (do this before inspecting any app)

Fetch all three pages in parallel and extract the rules listed below from each:

### 1a. Icon requirements
URL: `https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/design/design-teams-app-icon-store-appbar`

Extract:
- Required pixel dimensions for the **color** icon
- Required pixel dimensions for the **outline** icon
- Allowed file formats
- Maximum file sizes
- Any rules about transparency, background color, safe area, borders, or corner rounding

### 1b. Manifest schema
URL: `https://learn.microsoft.com/en-us/microsoftteams/platform/resources/schema/manifest-schema`

Extract:
- All required fields and their constraints (character limits, formats, allowed values)
- Any fields that must be absent or must not be set to an empty string
- Rules for `name`, `description`, `developer`, `icons`, `accentColor`, `validDomains`,
  `webApplicationInfo`, and `authorization`

### 1c. App package overview
URL: `https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/build-and-test/apps-package`

Extract:
- Required files in an app package
- Any additional constraints on icons not covered in 1a
- Package size limits

Apply **only** the rules you found in those pages. If a page does not specify a constraint,
do not invent one.

## Step 2 — Discover apps

List all subdirectories of `appstore/` (each is one app package):

```
find appstore -mindepth 1 -maxdepth 1 -type d | sort
```

For each subdirectory, proceed through steps 3–4. Run all apps in parallel.

## Step 3 — Validate each app

### Manifest

Read `<app-dir>/manifest.json` and check every field against the constraints you extracted
from the manifest schema in step 1b. Also run these cross-field consistency checks:

- Every hostname that appears in any `contentUrl` (across `staticTabs`, `configurableTabs`,
  `bots`, `messageExtensions`) must be present in `validDomains`.
- `webApplicationInfo.id` must match the app ID embedded in `webApplicationInfo.resource`.
- If `authorization.permissions.resourceSpecific` is present, each entry must have both
  `name` and `type` fields.

Use this snippet to measure field lengths (substitute the actual manifest path):

```bash
python3 - <<'EOF'
import json
with open('<manifest-path>') as f:
    m = json.load(f)
fields = [
    ('name.short',        m.get('name', {}).get('short', '')),
    ('name.full',         m.get('name', {}).get('full', '')),
    ('description.short', m.get('description', {}).get('short', '')),
    ('description.full',  m.get('description', {}).get('full', '')),
]
for name, value in fields:
    print(f'{name}: {len(value)} chars — "{value[:60]}{"..." if len(value) > 60 else ""}"')
EOF
```

### Icons

For each icon file referenced in `icons.color` and `icons.outline`, run:

```bash
file "<app-dir>/<icon-filename>"
ls -lh "<app-dir>/<icon-filename>"
```

Parse the pixel dimensions and file size from the output, then compare against the
requirements you extracted in step 1a.

## Step 4 — Produce the report

Print a single consolidated report:

---

## App Store Review

> Guidelines fetched from:
> - https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/design/design-teams-app-icon-store-appbar
> - https://learn.microsoft.com/en-us/microsoftteams/platform/resources/schema/manifest-schema
> - https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/build-and-test/apps-package

---

### <App Name> (`appstore/<dir>/`)

**Manifest version**: `<manifestVersion>` | **App version**: `<version>`

#### Manifest

| Check | Status | Detail |
|---|---|---|
| `name.short` | ✅ / ❌ | `<value>` (<N> chars, max per docs) |
| `name.full` | ✅ / ❌ | `<value>` (<N> chars, max per docs) |
| `description.short` | ✅ / ❌ | <N> chars (max per docs) |
| `description.full` | ✅ / ❌ | <N> chars (max per docs) |
| `developer.mpnId` | ✅ / ❌ | Absent / empty-string / valid |
| `developer URLs` | ✅ / ❌ | All HTTPS / issues listed |
| `validDomains` coverage | ✅ / ❌ | All contentUrl hosts covered / missing: [...] |
| `webApplicationInfo.resource` | ✅ / ❌ | Format correct / issue |
| `authorization` | ✅ / ❌ | Valid / issues |

#### Icons

| File | Required (per docs) | Actual | Status |
|---|---|---|---|
| `<color-icon-filename>` (color) | <requirement from docs> | <W>×<H> px, <format>, <size> | ✅ / ❌ |
| `<outline-icon-filename>` (outline) | <requirement from docs> | <W>×<H> px, <format>, <size> | ✅ / ❌ |

#### Issues to fix

Numbered list of all ❌ findings with specific remediation steps. If no issues, write
"No issues found — this package is ready for submission."

---

_(repeat for each app)_

---

### Summary

| App | Manifest | Icons | Ready? |
|---|---|---|---|
| <name> | ✅ / ❌ N issues | ✅ / ❌ N issues | Yes / No |

---

## Step 5 — Advisor review (uses claude-opus-4-8)

Before printing the final report, call the `advisor` tool. The advisor will review your
findings against the fetched guidelines and correct any misinterpretations. Incorporate
its feedback into the report before outputting it.

## Notes

- Use `file` for image dimensions — do not require Python Pillow or any external library.
- Do not modify any files; this is a read-only audit.
- If `manifest.json` is missing from an app directory, flag it as a blocking error and skip
  remaining checks for that app.
- If a documentation page is unreachable, note that in the report and skip the checks that
  depend on it rather than falling back to training-data assumptions.
