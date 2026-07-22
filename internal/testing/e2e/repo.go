// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// RepoRoot walks up from the test's working directory to the repository root,
// identified by config/target-allocator/clusterrole.yaml. It lets framework helpers
// reference shipped manifests regardless of the calling test package's depth.
func RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err, "getwd")
	for {
		if _, err := os.Stat(filepath.Join(dir, "config", "target-allocator", "clusterrole.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not locate repo root (config/target-allocator)")
		dir = parent
	}
}
