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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLeastWeightedTargetAllocator validates the least-weighted allocation strategy.
//
// The balanced-distribution property (targets split evenly across collectors) is
// inherently timing-dependent in e2e: whether both collectors are registered with
// the TA before the initial target assignment is a race between pod startup and TA
// reconciliation. That property is covered by unit tests in
// internal/allocation/least_weighted_test.go.
//
// This e2e test focuses on what IS reliably observable end-to-end:
//   - All targets are assigned when the strategy is least-weighted
//   - Existing assignments are preserved (not disrupted) when a new collector joins
//   - The new collector receives 0 targets on join (no rebalancing of sticky assignments)
func TestLeastWeightedTargetAllocator(t *testing.T) {
	env := newTestEnv(t)
	ctx, ns := env.ctx, env.ns

	taConfig := newTAConfig("least-weighted").withStaticTargets(testTargets).build()
	deployTA(t, ctx, ns, taConfig)
	waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

	deployCollectors(t, ctx, ns, 2)
	waitForStatefulSetReady(t, ctx, ns, "collector", 2)

	initialAssignment := waitForTargetDistribution(t, ctx, ns, "test-targets", 2)

	t.Run("all targets assigned", func(t *testing.T) {
		assert.Len(t, allAssignedTargets(initialAssignment), len(testTargets),
			"all targets should be assigned with least-weighted strategy")
		t.Logf("initial distribution: %v", countPerCollector(initialAssignment))
	})

	t.Run("existing assignments preserved on scale-up to 3", func(t *testing.T) {
		// Least-weighted does NOT rebalance existing targets when a new collector joins
		// (GetCollectorForTarget returns the current collector when it is still valid).
		// After scale-up: existing targets stay on their collectors, collector-2 starts at 0.
		scaleStatefulSet(t, ctx, ns, "collector", 3)
		waitForStatefulSetReady(t, ctx, ns, "collector", 3)

		afterScaleUp := waitForTargetDistribution(t, ctx, ns, "test-targets", 3)

		assert.Len(t, allAssignedTargets(afterScaleUp), len(testTargets),
			"all targets should remain assigned after scale-up")

		stayed := countStayedTargets(initialAssignment, afterScaleUp)
		assert.Equal(t, len(testTargets), stayed,
			"least-weighted should not move existing targets when a new collector joins")

		assert.Empty(t, afterScaleUp["collector-2"],
			"collector-2 should start with 0 targets (least-weighted only routes new targets there)")

		t.Logf("scale-up distribution: %v", countPerCollector(afterScaleUp))
	})
}

func countPerCollector(assignment map[string][]string) map[string]int {
	result := make(map[string]int, len(assignment))
	for c, targets := range assignment {
		result[c] = len(targets)
	}
	return result
}
