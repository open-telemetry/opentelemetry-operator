// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package e2e_ta_standalone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLeastWeightedTargetAllocator validates the least-weighted allocation strategy.
//
// Least-weighted assigns new targets to the collector with the fewest targets.
// Existing targets are NOT moved when a new collector joins (the strategy only
// applies at assignment time, not as a continuous rebalancer). Because tie-breaking
// uses Go map iteration order (non-deterministic), we only assert that the initial
// distribution is balanced (within ±1 per collector) rather than exact mapping.
func TestLeastWeightedTargetAllocator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ns := createTestNamespace(t, ctx)
	defer cleanupNamespace(t, ns)

	taConfig := buildStaticConfig("least-weighted", testTargets)
	deployTA(t, ctx, ns, taConfig)
	waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

	deployCollectors(t, ctx, ns, 2)
	waitForStatefulSetReady(t, ctx, ns, "collector", 2)

	// Wait for the TA to discover both collectors and produce a balanced assignment.
	// Least-weighted may initially assign all targets to collector-0 before collector-1
	// registers; waitForBalancedDistribution retries until every collector has ≥1 target.
	initialAssignment := waitForBalancedDistribution(t, ctx, ns, "test-targets", 2)

	t.Run("balanced initial distribution across 2 collectors", func(t *testing.T) {
		assertBalancedDistribution(t, initialAssignment, len(testTargets), 2)
	})

	t.Run("existing assignments preserved on scale-up to 3", func(t *testing.T) {
		// Least-weighted does NOT rebalance existing targets when a new collector joins
		// (GetCollectorForTarget returns the current collector if it is still valid).
		// After scale-up: existing targets stay, collector-2 starts at 0.
		scaleStatefulSet(t, ctx, ns, "collector", 3)
		waitForStatefulSetReady(t, ctx, ns, "collector", 3)

		afterScaleUp := waitForTargetDistribution(t, ctx, ns, "test-targets", 3)

		// Total must be preserved.
		assert.Len(t, allAssignedTargets(afterScaleUp), len(testTargets),
			"all targets should remain assigned after scale-up")

		// No target should move (stability property of least-weighted on scale-up).
		stayed := countStayedTargets(initialAssignment, afterScaleUp)
		assert.Equal(t, len(testTargets), stayed,
			"least-weighted should not move existing targets when a new collector joins")

		// New collector-2 has 0 targets; no rebalancing of already-assigned targets.
		assert.Empty(t, afterScaleUp["collector-2"],
			"collector-2 should start with 0 targets (least-weighted assigns only new targets to it)")

		t.Logf("scale-up distribution: %v", countPerCollector(afterScaleUp))
	})
}

// assertBalancedDistribution checks that:
//   - total assigned targets == expectedTotal
//   - each collector's count is within ±1 of the ideal even split
func assertBalancedDistribution(t *testing.T, assignment map[string][]string, expectedTotal, numCollectors int) {
	t.Helper()

	all := allAssignedTargets(assignment)
	assert.Len(t, all, expectedTotal, "total assigned targets should equal %d", expectedTotal)

	ideal := expectedTotal / numCollectors
	for collectorID, targets := range assignment {
		count := len(targets)
		assert.GreaterOrEqual(t, count, ideal-1,
			"collector %s has %d targets, expected at least %d (ideal %d ±1)",
			collectorID, count, ideal-1, ideal)
		assert.LessOrEqual(t, count, ideal+1,
			"collector %s has %d targets, expected at most %d (ideal %d ±1)",
			collectorID, count, ideal+1, ideal)
	}
	t.Logf("distribution: %v (ideal %d per collector)", countPerCollector(assignment), ideal)
}

func countPerCollector(assignment map[string][]string) map[string]int {
	result := make(map[string]int, len(assignment))
	for c, targets := range assignment {
		result[c] = len(targets)
	}
	return result
}
