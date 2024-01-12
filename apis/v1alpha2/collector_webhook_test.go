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

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		collector OpenTelemetryCollector
		warnings  []string
		err       string
	}{
		{
			name: "Test ",
			collector: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Config: Config{
						Processors: &AnyConfig{
							Object: map[string]interface{}{
								"batch": nil,
								"foo":   nil,
							},
						},
						Extensions: &AnyConfig{
							Object: map[string]interface{}{
								"foo": nil,
							},
						},
					},
				},
			},

			warnings: []string{
				"Collector config spec.config has null objects: extensions.foo:, processors.batch:, processors.foo:. For compatibility tooling (kustomize and kubectl edit) it is recommended to use empty obejects e.g. batch: {}.",
			},
		},
	}
	for _, tt := range tests {
		webhook := CollectorWebhook{}
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			warnings, err := webhook.validate(&tt.collector)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				assert.Equal(t, tt.err, err.Error())
			}
			assert.ElementsMatch(t, tt.warnings, warnings)
		})
	}
}
