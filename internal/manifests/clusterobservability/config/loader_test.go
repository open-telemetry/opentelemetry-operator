// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// asMap is a small helper to navigate the loosely-typed config maps produced by
// unmarshalling the embedded YAML.
func asMap(t *testing.T, v any, path string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.Truef(t, ok, "%s should be a map, got %T", path, v)
	return m
}

// The hostmetrics filesystem scraper must keep its exclude lists. On real nodes
// (e.g. OpenShift/RHCOS) the scraper walks the whole-root /hostfs mount, so
// without these excludes it would error on virtual/container-runtime
// filesystems. This path can't be exercised by the kind-based e2e (a kind node
// only has overlay/tmpfs/proc/sys mounts, all excluded), so it is guarded here.
func TestAgentBaseConfig_FilesystemScraperExcludes(t *testing.T) {
	base, err := NewConfigLoader().loadBaseConfig(AgentCollectorType)
	require.NoError(t, err)

	hostmetrics := asMap(t, base.Receivers["hostmetrics"], "receivers.hostmetrics")
	assert.Equal(t, "/hostfs", hostmetrics["root_path"],
		"hostmetrics must scrape the whole-root /hostfs mount")

	scrapers := asMap(t, hostmetrics["scrapers"], "hostmetrics.scrapers")
	filesystem := asMap(t, scrapers["filesystem"], "hostmetrics.scrapers.filesystem")

	excludeMounts := asMap(t, filesystem["exclude_mount_points"], "filesystem.exclude_mount_points")
	assert.Equal(t, "regexp", excludeMounts["match_type"])
	assert.NotEmpty(t, excludeMounts["mount_points"], "exclude_mount_points must list mount points")

	excludeFsTypes := asMap(t, filesystem["exclude_fs_types"], "filesystem.exclude_fs_types")
	assert.Equal(t, "strict", excludeFsTypes["match_type"])
	fsTypes, ok := excludeFsTypes["fs_types"].([]any)
	require.True(t, ok, "exclude_fs_types.fs_types should be a list")
	assert.Contains(t, fsTypes, "overlay",
		"overlay must be excluded so container-runtime filesystems are skipped")
}

// filelog must read pod logs (the run-as-root fix on OpenShift makes these
// readable); the include path and file-path attribute are asserted by the e2e
// via log.file.path, but the receiver config is locked in here too.
func TestAgentBaseConfig_FilelogReadsPodLogs(t *testing.T) {
	base, err := NewConfigLoader().loadBaseConfig(AgentCollectorType)
	require.NoError(t, err)

	filelog := asMap(t, base.Receivers["filelog"], "receivers.filelog")
	assert.Equal(t, true, filelog["include_file_path"],
		"include_file_path must be set so log.file.path is emitted")

	include, ok := filelog["include"].([]any)
	require.True(t, ok, "filelog.include should be a list")
	assert.Contains(t, include, "/var/log/pods/*/*/*.log",
		"filelog must collect pod logs from /var/log/pods")
}
