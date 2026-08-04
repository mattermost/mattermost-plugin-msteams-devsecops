// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"net/netip"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMattermostEnvKeys(t *testing.T) {
	environ := []string{
		"PATH=/usr/bin",
		"MM_FOO=bar",
		"TESTCONTAINERS_RYUK_DISABLED=true",
		"MM_BAR=",
		"NOT_MM_FOO=baz",
		"MALFORMED",
	}

	assert.Equal(t, []string{"MM_FOO", "MM_BAR"}, mattermostEnvKeys(environ))
}

func TestMattermostEnvKeysEmpty(t *testing.T) {
	assert.Empty(t, mattermostEnvKeys(nil))
	assert.Empty(t, mattermostEnvKeys([]string{"PATH=/usr/bin"}))
}

func TestSubnetCandidates(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		// Explicitly unset so a developer who exports the override in their shell
		// does not fail this case.
		t.Setenv(testSubnetEnvVar, "")

		candidates, err := testSubnetCandidates()
		require.NoError(t, err)
		require.Len(t, candidates, testSubnetCount)
		assert.Equal(t, netip.MustParsePrefix("192.168.222.0/24"), candidates[0])
		assert.Equal(t, netip.MustParsePrefix("192.168.231.0/24"), candidates[len(candidates)-1])
	})

	t.Run("valid override", func(t *testing.T) {
		t.Setenv(testSubnetEnvVar, "192.168.240.0/24")

		candidates, err := testSubnetCandidates()
		require.NoError(t, err)
		assert.Equal(t, []netip.Prefix{netip.MustParsePrefix("192.168.240.0/24")}, candidates)
	})

	t.Run("override with host bits is masked", func(t *testing.T) {
		t.Setenv(testSubnetEnvVar, "192.168.222.5/24")

		candidates, err := testSubnetCandidates()
		require.NoError(t, err)
		assert.Equal(t, []netip.Prefix{netip.MustParsePrefix("192.168.222.0/24")}, candidates)
	})

	t.Run("unparseable override", func(t *testing.T) {
		t.Setenv(testSubnetEnvVar, "nonsense")

		candidates, err := testSubnetCandidates()
		require.Error(t, err)
		assert.Nil(t, candidates)
		assert.Contains(t, err.Error(), testSubnetEnvVar)
	})

	t.Run("IPv6 override", func(t *testing.T) {
		t.Setenv(testSubnetEnvVar, "fd00::/64")

		candidates, err := testSubnetCandidates()
		require.Error(t, err)
		assert.Nil(t, candidates)
		assert.Contains(t, err.Error(), testSubnetEnvVar)
	})
}

func TestIsSubnetConflictErr(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "pool overlap",
			err:      errors.New("invalid pool request: Pool overlaps with other one on this address space"),
			expected: true,
		},
		{
			name:     "conflicting network",
			err:      errors.New("cannot create network 34e1e8b4 (br-34e1e8b4): conflicts with network 9f2c1a77 (br-9f2c1a77): networks have overlapping IPv4"),
			expected: true,
		},
		{
			name:     "daemon unreachable",
			err:      errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"),
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isSubnetConflictErr(tc.err))
		})
	}
}
