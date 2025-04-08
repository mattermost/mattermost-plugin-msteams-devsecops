// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package assets

import (
	_ "embed"
)

//go:embed mm-logo-color.png
var LogoColorData []byte

//go:embed mm-logo-outline.png
var LogoOutlineData []byte

//go:embed iframe.html.tmpl
var IFrameHTMLTemplate string

//go:embed appmanifest.json.tmpl
var AppManifestTemplate string
