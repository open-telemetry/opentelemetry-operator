// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"flag"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	FeatureGatesFlag = "feature-gates"
)

var (
	// CollectorUsesTargetAllocatorCR is the feature gate that enables the OpenTelemetryCollector reconciler to generate
	// TargetAllocator CRs instead of generating the manifests for its resources directly.
	CollectorUsesTargetAllocatorCR = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.targetallocatorcr",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("causes collector reconciliation to create a target allocator CR instead of creating resources directly"),
		featuregate.WithRegisterFromVersion("v0.112.0"),
	)
	// EnableNativeSidecarContainers is the feature gate that controls whether a
	// sidecar should be injected as a native sidecar or the classic way.
	// Native sidecar containers have been available since kubernetes v1.28 in
	// alpha and v1.29 in beta.
	// It needs to be enabled with +featureGate=SidecarContainers.
	// See:
	// https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-gates-for-alpha-or-beta-features
	EnableNativeSidecarContainers = featuregate.GlobalRegistry().MustRegister(
		"operator.sidecarcontainers.native",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("controls whether the operator supports sidecar containers as init containers. Should only be enabled on k8s v1.29+"),
		featuregate.WithRegisterFromVersion("v0.111.0"),
	)
	// PrometheusOperatorIsAvailable is the feature gate that enables features associated to the Prometheus Operator.
	PrometheusOperatorIsAvailable = featuregate.GlobalRegistry().MustRegister(
		"operator.observability.prometheus",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables features associated to the Prometheus Operator"),
		featuregate.WithRegisterFromVersion("v0.82.0"),
	)
	// SetGolangFlags is the feature gate that enables automatically setting GOMEMLIMIT and GOMAXPROCS for the
	// collector, bridge, and target allocator.
	SetGolangFlags = featuregate.GlobalRegistry().MustRegister(
		"operator.golang.flags",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables feature to set GOMEMLIMIT and GOMAXPROCS automatically"),
		featuregate.WithRegisterFromVersion("v0.100.0"),
	)
	// EnableTargetAllocatorMTLS is the feature gate that enables mTLS between the target allocator and the collector.
	EnableTargetAllocatorMTLS = featuregate.GlobalRegistry().MustRegister(
		"operator.targetallocator.mtls",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables mTLS between the target allocator and the collector"),
		featuregate.WithRegisterFromVersion("v0.111.0"),
	)
	// EnableTargetAllocatorFallbackStrategy is the feature gate that enables consistent-hashing as the fallback
	// strategy for allocation strategies that might not assign all jobs (per-node).
	EnableTargetAllocatorFallbackStrategy = featuregate.GlobalRegistry().MustRegister(
		"operator.targetallocator.fallbackstrategy",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables fallback allocation strategy for the target allocator"),
		featuregate.WithRegisterFromVersion("v0.114.0"),
	)
	// EnableConfigDefaulting is the feature gate that enables the operator to default the endpoint for known components.
	EnableConfigDefaulting = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.default.config",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables the operator to default the endpoint for known components"),
		featuregate.WithRegisterFromVersion("v0.110.0"),
	)
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	reg.RegisterFlags(flagSet)
	return flagSet
}
