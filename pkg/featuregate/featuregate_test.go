// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"
)

const (
	basicGate    = "basic"
	advancedGate = "advanced"
	falseGate    = "false"
)

func TestSetFlag(t *testing.T) {
	featuregate.GlobalRegistry().MustRegister(basicGate, featuregate.StageAlpha)
	featuregate.GlobalRegistry().MustRegister(advancedGate, featuregate.StageBeta)
	featuregate.GlobalRegistry().MustRegister(falseGate, featuregate.StageStable, featuregate.WithRegisterToVersion("v0.0.1"))
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
