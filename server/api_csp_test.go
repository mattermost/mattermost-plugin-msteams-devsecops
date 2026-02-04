// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeForLog(t *testing.T) {
	t.Run("replaces newlines and tabs with space", func(t *testing.T) {
		in := "line1\nline2\rline3\t tab"
		out := sanitizeForLog(in)
		assert.NotContains(t, out, "\n")
		assert.NotContains(t, out, "\r")
		assert.NotContains(t, out, "\t")
		assert.Equal(t, "line1 line2 line3   tab", out)
	})

	t.Run("truncates at maxCSPReportFieldLen runes and appends ellipsis", func(t *testing.T) {
		in := strings.Repeat("a", maxCSPReportFieldLen+100)
		out := sanitizeForLog(in)
		assert.Equal(t, maxCSPReportFieldLen+3, len([]rune(out)), "output should be 500 runes + \"...\"")
		assert.True(t, strings.HasSuffix(out, "..."))
	})

	t.Run("preserves short normal string", func(t *testing.T) {
		in := "https://example.com/script.js"
		assert.Equal(t, in, sanitizeForLog(in))
	})

	t.Run("replaces non-printable runes with space", func(t *testing.T) {
		in := "foo\x00bar"
		out := sanitizeForLog(in)
		assert.Equal(t, "foo bar", out)
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		assert.Empty(t, sanitizeForLog(""))
	})

	t.Run("preserves Unicode runes", func(t *testing.T) {
		in := "café résumé"
		assert.Equal(t, in, sanitizeForLog(in))
	})

	t.Run("truncation does not split multi-byte rune", func(t *testing.T) {
		// 501 runes: 500 'a' + 1 'é' (2 bytes in UTF-8). Truncate to 500 runes -> "aaa...aaa" no é.
		in := strings.Repeat("a", maxCSPReportFieldLen) + "é"
		out := sanitizeForLog(in)
		assert.True(t, strings.HasSuffix(out, "..."), "should be truncated with ellipsis")
		assert.LessOrEqual(t, len([]rune(out)), maxCSPReportFieldLen+3)
	})
}
