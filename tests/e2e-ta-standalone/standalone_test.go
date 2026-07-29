// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tastandalone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-telemetry/opentelemetry-operator/internal/testing/e2e"
)

// TestStandaloneTargetAllocator validates the standalone TA with consistent-hashing:
// target distribution, scale-up (2→3) and scale-down (3→2), and HTTP API contract.
func TestStandaloneTargetAllocator(t *testing.T) {
	type initialAssignmentKey struct{}

	feat := features.New("standalone TA with consistent-hashing strategy").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var ns string
			ctx, ns = setupTestNamespace(ctx, t, cfg)

			taConfig := newTAConfig("consistent-hashing").withStaticTargets(testTargets).build()
			deployTA(t, ctx, cfg, ns, taConfig)
			e2e.WaitForDeployment(ctx, t, cfg, ns, "target-allocator", testTimeout)

			deployCollectors(t, ctx, cfg, ns, 2)
			e2e.WaitForStatefulSet(ctx, t, cfg, ns, "collector", 2, testTimeout)

			assignment := waitForTargetDistribution(t, ctx, cfg, ns, "test-targets", 2)
			return context.WithValue(ctx, initialAssignmentKey{}, assignment)
		}).
		Assess("targets distributed across collectors", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			assignment := ctx.Value(initialAssignmentKey{}).(map[string][]string)
			allTargets := allAssignedTargets(assignment)
			assert.Len(t, allTargets, len(testTargets), "all targets should be assigned")
			hasTargets := false
			for _, targets := range assignment {
				if len(targets) > 0 {
					hasTargets = true
					break
				}
			}
			assert.True(t, hasTargets, "at least one collector should have targets")
			return ctx
		}).
		Assess("scale up preserves consistency", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ns := nsFromCtx(ctx)
			initialAssignment := ctx.Value(initialAssignmentKey{}).(map[string][]string)

			scaleStatefulSet(t, ctx, cfg, ns, "collector", 3)
			e2e.WaitForStatefulSet(ctx, t, cfg, ns, "collector", 3, testTimeout)

			afterScaleUp := waitForTargetDistribution(t, ctx, cfg, ns, "test-targets", 3)
			assert.Len(t, allAssignedTargets(afterScaleUp), len(testTargets), "all targets assigned after scale-up")

			stayed := countStayedTargets(initialAssignment, afterScaleUp)
			assert.GreaterOrEqual(t, stayed, 3, "at least 3/6 targets should stay (consistent hashing)")
			return ctx
		}).
		Assess("scale down reassigns targets", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ns := nsFromCtx(ctx)

			scaleStatefulSet(t, ctx, cfg, ns, "collector", 2)
			e2e.WaitForStatefulSet(ctx, t, cfg, ns, "collector", 2, testTimeout)

			afterScaleDown := waitForTargetDistribution(t, ctx, cfg, ns, "test-targets", 2)
			assert.Len(t, allAssignedTargets(afterScaleDown), len(testTargets), "all targets assigned after scale-down")
			assert.Empty(t, afterScaleDown["collector-2"], "collector-2 should have no targets after scale-down")
			return ctx
		}).
		Assess("HTTP API contract", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ns := nsFromCtx(ctx)
			proxyBase := taProxyBase(ns)

			body := kubectlGetRaw(t, ctx, cfg, proxyBase+"/jobs")
			assert.Contains(t, string(body), "test-targets", "/jobs should list test-targets")

			body = kubectlGetRaw(t, ctx, cfg, proxyBase+"/scrape_configs")
			assert.Contains(t, string(body), "test-targets", "/scrape_configs should list test-targets")

			kubectlGetRaw(t, ctx, cfg, proxyBase+"/livez")
			kubectlGetRaw(t, ctx, cfg, proxyBase+"/readyz")

			body, err := e2e.ClientSet(t, cfg).CoreV1().RESTClient().Get().
				AbsPath(proxyBase+"/jobs/test-targets/targets").
				Param("collector_id", "nonexistent").
				DoRaw(ctx)
			require.NoError(t, err)
			assert.NotContains(t, string(body), "target-a:8080", "unknown collector should have no targets")
			return ctx
		}).
		Feature()

	testenv.Test(t, feat)
}
