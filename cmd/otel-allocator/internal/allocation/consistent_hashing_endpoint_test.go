// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"maps"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

func TestConsistentHashingEndpointRelativelyEvenDistribution(t *testing.T) {
	numCols := 15
	numItems := 10000
	cols := MakeNCollectors(numCols, 0)
	expectedPerCollector := float64(numItems / numCols)
	expectedDelta := (expectedPerCollector * 1.5) - expectedPerCollector
	c, _ := New(consistentHashingEndpointStrategyName, logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargets(numItems, 0, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, numItems)
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	for _, col := range actualCollectors {
		assert.InDelta(t, col.NumTargets, expectedPerCollector, expectedDelta)
	}
}

func TestTargetsWithNoCollectorsConsistentHashingEndpoint(t *testing.T) {
	c, _ := New(consistentHashingEndpointStrategyName, logger)

	numItems := 10
	c.SetTargets(MakeNNewTargetsWithEmptyCollectors(numItems, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, numItems)

	numCols := 2
	cols := MakeNCollectors(numCols, 0)
	c.SetCollectors(cols)
	expectedPerCollector := float64(numItems / numCols)
	expectedDelta := (expectedPerCollector * 1.5) - expectedPerCollector
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	for _, col := range actualCollectors {
		assert.InDelta(t, col.NumTargets, expectedPerCollector, expectedDelta)
	}
}

// TestConsistentHashingEndpointSpreadsSameAddress verifies that, unlike
// consistent-hashing, this strategy distributes targets that share a
// host:port but differ by metrics path or params across multiple
// collectors, instead of co-locating them.
func TestConsistentHashingEndpointSpreadsSameAddress(t *testing.T) {
	const addr = "10.0.0.1:9001"
	cols := MakeNCollectors(10, 0)

	mkItem := func(jobName, metricsPath, paramShard string) *target.Item {
		ls := labels.FromMap(map[string]string{
			"__address__":      addr,
			"__metrics_path__": metricsPath,
			"__param_shard":    paramShard,
		})
		return target.NewItem(jobName, addr, ls, "", target.HashLabels(ls, jobName))
	}

	items := []*target.Item{
		mkItem("group-a-0", "/metrics/group_a", "0"),
		mkItem("group-a-1", "/metrics/group_a", "1"),
		mkItem("group-b-hf", "/metrics/group_b_hf", ""),
		mkItem("group-b-lf", "/metrics/group_b_lf", ""),
	}

	s := newConsistentHashingEndpointStrategy().(*consistentHashingEndpointStrategy)
	s.SetCollectors(cols)
	seen := map[string]struct{}{}
	for _, item := range items {
		collector, err := s.GetCollectorForTarget(cols, item)
		require.NoError(t, err)
		seen[collector.Name] = struct{}{}
	}
	assert.Greater(t, len(seen), 1, "must spread same-address targets across collectors")
}

// TestConsistentHashingEndpointIgnoresNonURLLabels verifies the anti-churn
// guarantee: the hash key is built only from the scrape endpoint, so
// changing a label that does not affect the URL must not move the target to
// a different collector.
func TestConsistentHashingEndpointIgnoresNonURLLabels(t *testing.T) {
	const addr = "10.0.0.1:9001"
	cols := MakeNCollectors(10, 0)

	baseLabels := map[string]string{
		"__address__":      addr,
		"__scheme__":       "http",
		"__metrics_path__": "/metrics",
		"__param_shard":    "0",
	}
	withExtra := func(name, value string) *target.Item {
		ls := map[string]string{}
		maps.Copy(ls, baseLabels)
		ls[name] = value
		itemLabels := labels.FromMap(ls)
		return target.NewItem("job", addr, itemLabels, "", target.HashLabels(itemLabels, "job"))
	}

	s := newConsistentHashingEndpointStrategy().(*consistentHashingEndpointStrategy)
	s.SetCollectors(cols)

	// Same scrape endpoint, different metadata labels (e.g. instance
	// metadata that changes over a target's lifetime).
	before, err := s.GetCollectorForTarget(cols, withExtra("some_metadata_label", "a1"))
	require.NoError(t, err)
	after, err := s.GetCollectorForTarget(cols, withExtra("some_metadata_label", "b2"))
	require.NoError(t, err)

	assert.Equal(t, before.Name, after.Name,
		"must not reassign a target when a non-URL label changes")
}
