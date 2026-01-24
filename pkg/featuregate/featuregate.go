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
		featuregate.StageStable,
		featuregate.WithRegisterDescription("enables the operator to default the endpoint for known components"),
		featuregate.WithRegisterFromVersion("v0.110.0"),
		featuregate.WithRegisterToVersion("v0.139.0"),
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
