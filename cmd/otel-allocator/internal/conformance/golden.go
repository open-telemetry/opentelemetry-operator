// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

// goldenTarget mirrors one element of `promtool check service-discovery` output
// (promtool's own sdCheckResult, hence labels.Labels). An empty Labels set means
// Prometheus dropped the target during relabeling.
type goldenTarget struct {
	DiscoveredLabels labels.Labels `json:"discoveredLabels"`
	Labels           labels.Labels `json:"labels"`
}

func (g goldenTarget) dropped() bool { return g.Labels.IsEmpty() }

func (g goldenTarget) key() targetKey {
	return targetKey{job: g.DiscoveredLabels.Get("job"), address: g.DiscoveredLabels.Get("__address__")}
}

// loadGolden reads a committed promtool golden file, bucketed by (job, address).
func loadGolden(t *testing.T, path string) map[targetKey][]goldenTarget {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "reading golden %s (run `make ta-conformance-regen` or `go test -update`)", path)
	var targets []goldenTarget
	require.NoErrorf(t, json.Unmarshal(data, &targets), "parsing golden %s", path)
	out := make(map[targetKey][]goldenTarget, len(targets))
	for _, gt := range targets {
		out[gt.key()] = append(out[gt.key()], gt)
	}
	return out
}

// regenGolden (re)generates a golden file by running promtool against the
// fixture, once per job, concatenating the results. The promtool binary is
// taken from $PROMTOOL (default "promtool"); it must match the Prometheus
// version in go.mod.
func regenGolden(t *testing.T, promYAMLPath string, jobs []string, outPath string) {
	t.Helper()
	promtool := os.Getenv("PROMTOOL")
	if promtool == "" {
		promtool = "promtool"
	}
	// static_configs / file_sd resolve immediately; promtool waits the full
	// --timeout before dumping, so a short timeout keeps regeneration fast.
	timeout := os.Getenv("PROMTOOL_TIMEOUT")
	if timeout == "" {
		timeout = "2s"
	}
	var all []goldenTarget
	for _, job := range jobs {
		out, err := exec.CommandContext( //nolint:gosec,G702 // test code, not actual command injection
			t.Context(),
			promtool,
			"check",
			"service-discovery",
			"--timeout="+timeout,
			promYAMLPath,
			job,
		).Output()
		require.NoErrorf(t, err, "promtool check service-discovery for job %q (stderr may have detail)", job)
		var targets []goldenTarget
		require.NoErrorf(t, json.Unmarshal(out, &targets), "parsing promtool output for job %q", job)
		all = append(all, targets...)
	}
	data, err := json.MarshalIndent(all, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outPath, append(data, '\n'), 0o600))
}
