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
	"github.com/stretchr/testify/require"
)

// TestStandaloneTargetAllocator validates the standalone TA with consistent-hashing:
// target distribution, scale-up (2→3) and scale-down (3→2), and HTTP API contract.
func TestStandaloneTargetAllocator(t *testing.T) {
	env := newTestEnv(t)
	ctx, ns := env.ctx, env.ns

	taConfig := newTAConfig("consistent-hashing").withStaticTargets(testTargets).build()
	deployTA(t, ctx, ns, taConfig)
	waitForDeploymentReady(t, ctx, ns, "target-allocator", 1)

	deployCollectors(t, ctx, ns, 2)
	waitForStatefulSetReady(t, ctx, ns, "collector", 2)

	initialAssignment := waitForTargetDistribution(t, ctx, ns, "test-targets", 2)

	t.Run("targets distributed across collectors", func(t *testing.T) {
		allTargets := allAssignedTargets(initialAssignment)
		assert.Len(t, allTargets, len(testTargets), "all targets should be assigned")
		hasTargets := false
		for _, targets := range initialAssignment {
			if len(targets) > 0 {
				hasTargets = true
				break
			}
		}
		assert.True(t, hasTargets, "at least one collector should have targets")
	})

	t.Run("scale up preserves consistency", func(t *testing.T) {
		scaleStatefulSet(t, ctx, ns, "collector", 3)
		waitForStatefulSetReady(t, ctx, ns, "collector", 3)

		afterScaleUp := waitForTargetDistribution(t, ctx, ns, "test-targets", 3)
		assert.Len(t, allAssignedTargets(afterScaleUp), len(testTargets), "all targets assigned after scale-up")

		stayed := countStayedTargets(initialAssignment, afterScaleUp)
		assert.GreaterOrEqual(t, stayed, 3, "at least 3/6 targets should stay (consistent hashing)")
	})

	t.Run("scale down reassigns targets", func(t *testing.T) {
		scaleStatefulSet(t, ctx, ns, "collector", 2)
		waitForStatefulSetReady(t, ctx, ns, "collector", 2)

		afterScaleDown := waitForTargetDistribution(t, ctx, ns, "test-targets", 2)
		assert.Len(t, allAssignedTargets(afterScaleDown), len(testTargets), "all targets assigned after scale-down")
		assert.Empty(t, afterScaleDown["collector-2"], "collector-2 should have no targets after scale-down")
	})

	t.Run("HTTP API contract", func(t *testing.T) {
		proxyBase := taProxyBase(ns)

		body := kubectlGetRaw(t, ctx, proxyBase+"/jobs")
		assert.Contains(t, string(body), "test-targets", "/jobs should list test-targets")

		body = kubectlGetRaw(t, ctx, proxyBase+"/scrape_configs")
		assert.Contains(t, string(body), "test-targets", "/scrape_configs should list test-targets")

		kubectlGetRaw(t, ctx, proxyBase+"/livez")
		kubectlGetRaw(t, ctx, proxyBase+"/readyz")

		body, err := clientset.CoreV1().RESTClient().Get().
			AbsPath(proxyBase + "/jobs/test-targets/targets").
			Param("collector_id", "nonexistent").
			DoRaw(ctx)
		require.NoError(t, err)
		assert.NotContains(t, string(body), "target-a:8080", "unknown collector should have no targets")
	})
}
