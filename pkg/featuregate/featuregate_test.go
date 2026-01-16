// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"
	"k8s.io/apimachinery/pkg/util/version"

	operatorversion "github.com/open-telemetry/opentelemetry-operator/internal/version"
)

const (
	basicGate    = "basic"
	advancedGate = "advanced"
	falseGate    = "false"
)

func TestSetFlag(t *testing.T) {
	featuregate.GlobalRegistry().MustRegister(basicGate, featuregate.StageAlpha)
	featuregate.GlobalRegistry().MustRegister(advancedGate, featuregate.StageBeta)
	// Use a far-future version to avoid triggering the lifecycle test
	featuregate.GlobalRegistry().MustRegister(falseGate, featuregate.StageStable, featuregate.WithRegisterToVersion("v999.0.0"))
	tests := []struct {
		name          string
		args          []string
		expectedTrue  []string
		expectedFalse []string
		expectedErr   string
	}{
		{
			name:         "simple set",
			args:         []string{"--feature-gates=basic"},
			expectedTrue: []string{basicGate},
		},
		{
			name:         "two flags",
			args:         []string{"--feature-gates=basic,advanced"},
			expectedTrue: []string{basicGate, advancedGate},
		},
		{
			name:          "one true one false",
			args:          []string{"--feature-gates=basic,-advanced"},
			expectedTrue:  []string{basicGate},
			expectedFalse: []string{advancedGate},
		},
		{
			name:        "invalid set",
			args:        []string{"--feature-gates=01"},
			expectedErr: `no such feature gate -01`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flgs := Flags(featuregate.GlobalRegistry())
			err := flgs.Parse(tt.args)
			if tt.expectedErr != "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			featuregate.GlobalRegistry().VisitAll(func(gate *featuregate.Gate) {
				for _, id := range tt.expectedTrue {
					if gate.ID() == id {
						assert.True(t, gate.IsEnabled())
					}
				}
				for _, id := range tt.expectedFalse {
					if gate.ID() == id {
						assert.False(t, gate.IsEnabled())
					}
				}
			})
		})
	}
}

// TestFeatureGateLifecycle ensures that feature gates with a ToVersion are removed
// when the operator version meets or exceeds that version. This prevents stale
// feature gates from accumulating in the codebase.
func TestFeatureGateLifecycle(t *testing.T) {
	// Get operator version from build-injected version (set via -ldflags during make)
	operatorVersion := operatorversion.Get().Operator
	if operatorVersion == "" {
		t.Skip("Operator version not set (run with 'make test' or set via -ldflags)")
	}

	// Ensure version starts with 'v' for semantic version parsing
	if !strings.HasPrefix(operatorVersion, "v") {
		operatorVersion = "v" + operatorVersion
	}

	var violations []string

	// Check all registered feature gates
	featuregate.GlobalRegistry().VisitAll(func(gate *featuregate.Gate) {
		toVersion := gate.ToVersion()

		// Skip gates without a ToVersion set
		if toVersion == "" {
			return
		}

		// Parse versions
		currentVer, err := version.ParseSemantic(operatorVersion)
		if err != nil {
			t.Fatalf("Failed to parse operator version %q: %v", operatorVersion, err)
		}

		toVer, err := version.ParseSemantic(toVersion)
		if err != nil {
			// Skip if ToVersion is not parseable (might be malformed)
			t.Logf("Warning: feature gate %q has unparseable ToVersion %q", gate.ID(), toVersion)
			return
		}

		// Check if we've reached or exceeded the ToVersion
		if currentVer.AtLeast(toVer) {
			violations = append(violations, fmt.Sprintf(
				"Feature gate %q (stage=%v) has reached its ToVersion %q (current: %v).\n"+
					"  Action required:\n"+
					"  1. If stable: Remove all IsEnabled() checks and assume it's always on\n"+
					"  2. Then remove the MustRegister() call from featuregate.go\n"+
					"  3. Or update ToVersion to a future version if not ready to remove",
				gate.ID(), gate.Stage(), toVersion, operatorVersion,
			))
		}
	})

	// Fail the test if any violations were found
	if len(violations) > 0 {
		t.Fatalf("Found %d feature gate(s) that should be removed:\n\n%s",
			len(violations), strings.Join(violations, "\n\n"))
	}
}

func TestApplyFeatureGateOverrides_EmptyString(t *testing.T) {
	// Test that empty string is handled gracefully
	err := ApplyFeatureGateOverrides("")
	require.NoError(t, err)
}

func TestApplyFeatureGateOverrides_SingleGate(t *testing.T) {
	// Test enabling a single feature gate
	registry := featuregate.GlobalRegistry()

	// Get original state
	originalState := SetGolangFlags.IsEnabled()
	t.Cleanup(func() {
		// Restore original state
		_ = registry.Set(SetGolangFlags.ID(), originalState)
	})

	// Apply feature gate
	err := ApplyFeatureGateOverrides("operator.golang.flags")
	require.NoError(t, err)

	// Verify it was enabled
	assert.True(t, SetGolangFlags.IsEnabled())
}

func TestApplyFeatureGateOverrides_MultipleGates(t *testing.T) {
	// Test enabling multiple feature gates
	registry := featuregate.GlobalRegistry()

	// Get original states
	originalGolangFlags := SetGolangFlags.IsEnabled()
	originalMTLS := EnableTargetAllocatorMTLS.IsEnabled()
	t.Cleanup(func() {
		// Restore original states
		_ = registry.Set(SetGolangFlags.ID(), originalGolangFlags)
		_ = registry.Set(EnableTargetAllocatorMTLS.ID(), originalMTLS)
	})

	// Apply multiple feature gates
	err := ApplyFeatureGateOverrides("operator.golang.flags,operator.targetallocator.mtls")
	require.NoError(t, err)

	// Verify both were enabled
	assert.True(t, SetGolangFlags.IsEnabled())
	assert.True(t, EnableTargetAllocatorMTLS.IsEnabled())
}

func TestApplyFeatureGateOverrides_MixedEnableDisable(t *testing.T) {
	// Test enabling some gates and disabling others
	registry := featuregate.GlobalRegistry()

	// Get original states
	originalGolangFlags := SetGolangFlags.IsEnabled()
	originalMTLS := EnableTargetAllocatorMTLS.IsEnabled()
	originalNetworkPolicy := EnableOperatorNetworkPolicy.IsEnabled()
	t.Cleanup(func() {
		// Restore original states
		_ = registry.Set(SetGolangFlags.ID(), originalGolangFlags)
		_ = registry.Set(EnableTargetAllocatorMTLS.ID(), originalMTLS)
		_ = registry.Set(EnableOperatorNetworkPolicy.ID(), originalNetworkPolicy)
	})

	// Apply mixed gates: enable golang.flags and mtls, disable networkpolicy
	err := ApplyFeatureGateOverrides("operator.golang.flags,operator.targetallocator.mtls,-operator.networkpolicy")
	require.NoError(t, err)

	// Verify states
	assert.True(t, SetGolangFlags.IsEnabled(), "golang.flags should be enabled")
	assert.True(t, EnableTargetAllocatorMTLS.IsEnabled(), "mtls should be enabled")
	assert.False(t, EnableOperatorNetworkPolicy.IsEnabled(), "networkpolicy should be disabled")
}
