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

package featuregate

import (
	"testing"

	"go.opentelemetry.io/collector/featuregate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
