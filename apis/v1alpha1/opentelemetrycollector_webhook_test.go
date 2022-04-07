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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOTELColDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)
	tests := []struct {
		name     string
		otelcol  OpenTelemetryCollector
		expected OpenTelemetryCollector
	}{
		{
			name:    "all fields default",
			otelcol: OpenTelemetryCollector{},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "provided values in spec",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.otelcol.Default()
			assert.Equal(t, test.expected, test.otelcol)
		})
	}
}
