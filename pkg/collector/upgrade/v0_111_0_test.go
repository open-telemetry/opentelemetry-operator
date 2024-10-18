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

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_111_0Upgrade(t *testing.T) {

	defaultCollector := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-my-instance",
			Namespace: "somewhere",
		},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			Version: "0.110.0",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{},
			Config:                    v1beta1.Config{},
		},
	}

	defaultCollectorWithConfig := defaultCollector.DeepCopy()

	defaultCollectorWithConfig.Spec.Config.Service.Telemetry = &v1beta1.AnyConfig{
		Object: map[string]interface{}{
			"metrics": map[string]interface{}{
				"address": "1.2.3.4:8888",
			},
		},
	}

	tt := []struct {
		name     string
		input    v1beta1.OpenTelemetryCollector
		expected v1beta1.OpenTelemetryCollector
	}{
		{
			name:     "telemetry settings exist",
			input:    *defaultCollectorWithConfig,
			expected: *defaultCollectorWithConfig,
		},
		{
			name:  "telemetry settings do not exist",
			input: *defaultCollector.DeepCopy(),
			expected: func() v1beta1.OpenTelemetryCollector {
				col := defaultCollector.DeepCopy()
				col.Spec.Config.Service.Telemetry = &v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"metrics": map[string]interface{}{
							"address": "0.0.0.0:8888",
						},
					},
				}
				return *col
			}(),
		},
	}

	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.111.0"),
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			col, err := versionUpgrade.ManagedInstance(context.Background(), tc.input)
			if err != nil {
				t.Errorf("expect err: nil but got: %v", err)
			}
			assert.Equal(t, tc.expected.Spec.Config.Service.Telemetry, col.Spec.Config.Service.Telemetry)
		})
	}
}
