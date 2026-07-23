// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"flag"
	"fmt"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	FeatureGatesFlag = "feature-gates"
)

var (
	// SetGolangFlags is the feature gate that enables automatically setting GOMEMLIMIT and GOMAXPROCS for the
	// collector, bridge, and target allocator.
	SetGolangFlags = featuregate.GlobalRegistry().MustRegister(
		"operator.golang.flags",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables feature to set GOMEMLIMIT and GOMAXPROCS automatically"),
		featuregate.WithRegisterFromVersion("v0.100.0"),
	)
	// EnableTargetAllocatorFallbackStrategy is the feature gate that enables consistent-hashing as the fallback
	// strategy for allocation strategies that might not assign all jobs (per-node).
	EnableTargetAllocatorFallbackStrategy = featuregate.GlobalRegistry().MustRegister(
		"operator.targetallocator.fallbackstrategy",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables fallback allocation strategy for the target allocator"),
		featuregate.WithRegisterFromVersion("v0.114.0"),
	)
	// EnableOperatorNetworkPolicy is the feature gate that enables the operator to create network policies for the operator.
	EnableOperatorNetworkPolicy = featuregate.GlobalRegistry().MustRegister(
		"operator.networkpolicy",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables the operator to create network policies for the operator"),
		featuregate.WithRegisterFromVersion("v0.132.0"),
	)
	// EnableOperandNetworkPolicy is the feature gate that enables the operator to create network policies for the collector.
	EnableOperandNetworkPolicy = featuregate.GlobalRegistry().MustRegister(
		"operand.networkpolicy",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables the operator to create network policies for operands,  collector and target allocator are supported"),
	)
	// EnableClusterObservability is the feature gate that enables the ClusterObservability controller.
	EnableClusterObservability = featuregate.GlobalRegistry().MustRegister(
		"operator.clusterobservability",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables the ClusterObservability controller for managed observability deployment"),
		featuregate.WithRegisterFromVersion("v0.134.0"),
	)
	// EnableInstrumentationInjector is the feature gate that switches auto-instrumentation
	// activation to the opentelemetry-injector (https://github.com/open-telemetry/opentelemetry-injector).
	// Instead of setting runtime-specific environment variables (e.g. JAVA_TOOL_OPTIONS) directly on
	// containers, the injector shared object is loaded into application processes via LD_PRELOAD and
	// sets those variables in-process. Currently Java, .NET, Node.js and Python are supported.
	//
	// The injector is only published for amd64 and arm64. The operator cannot reliably determine the
	// target architecture of a pod at admission time, so with the gate enabled, pods scheduled on
	// other architectures are not instrumented (and for Java and Python fail to start, as their
	// instrumentation init containers reference the missing injector). The gate must not graduate
	// before the injector covers all architectures the auto-instrumentation images are built for.
	EnableInstrumentationInjector = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.injector",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("use the opentelemetry-injector LD_PRELOAD object to activate auto-instrumentation instead of setting runtime-specific environment variables directly (currently Java, .NET, Node.js and Python)"),
		featuregate.WithRegisterFromVersion("v0.155.0"),
	)
	// UseCollectorDefaultTelemetryShape, when enabled (default at beta), makes
	// the operator-injected Prometheus telemetry reader use collector defaults
	// for without_type_suffix, without_units, and without_scope_info — metric
	// names emitted by operator-managed collectors no longer carry type
	// suffixes, units, or scope_info. When disabled, the operator explicitly
	// sets all three to false to preserve the pre-v0.154.0 metric name shape.
	// See open-telemetry/opentelemetry-operator#5075.
	UseCollectorDefaultTelemetryShape = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.usedefaulttelemetryshape",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("when enabled (default), the operator-injected Prometheus telemetry reader uses collector defaults for without_type_suffix/without_units/without_scope_info. When disabled, the operator explicitly sets all three to false to preserve the pre-v0.154.0 metric name shape."),
		featuregate.WithRegisterFromVersion("v0.152.0"),
	)
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	reg.RegisterFlags(flagSet)
	return flagSet
}

// ApplyFeatureGateOverrides applies feature gate configuration from a comma-separated string.
// Format matches CLI flag: "gate1,gate2,-gate3" where - prefix disables the gate.
// This is needed because feature gates are stored in GlobalRegistry(), not in Config struct.
func ApplyFeatureGateOverrides(gates string) error {
	if gates == "" {
		return nil
	}

	// Create temporary FlagSet to apply feature gates
	fs := flag.NewFlagSet("config-overrides", flag.ContinueOnError)
	reg := featuregate.GlobalRegistry()
	reg.RegisterFlags(fs)

	// Apply the gates string to the global registry
	if err := fs.Set(FeatureGatesFlag, gates); err != nil {
		return fmt.Errorf("failed to apply feature gates: %w", err)
	}

	return nil
}
