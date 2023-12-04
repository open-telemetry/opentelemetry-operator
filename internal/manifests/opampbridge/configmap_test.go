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

package opampbridge

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"

	"github.com/stretchr/testify/assert"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLables := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.69.0",
	}

	t.Run("should return expected opamp-bridge config map", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-opamp-bridge"
		expectedLables["app.kubernetes.io/name"] = "my-instance-opamp-bridge"

		expectedData := map[string]string{
			"remoteconfiguration.yaml": `capabilities:
  AcceptsOpAMPConnectionSettings: true
  AcceptsOtherConnectionSettings: true
  AcceptsRemoteConfig: true
  AcceptsRestartCommand: true
  ReportsEffectiveConfig: true
  ReportsHealth: true
  ReportsOwnLogs: true
  ReportsOwnMetrics: true
  ReportsOwnTraces: true
  ReportsRemoteConfig: true
  ReportsStatus: true
componentsAllowed:
  exporters:
  - logging
  processors:
  - memory_limiter
  receivers:
  - otlp
endpoint: ws://opamp-server:4320/v1/opamp
headers:
  authorization: access-12345-token
`}

		opampBridge := v1alpha1.OpAMPBridge{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-namespace",
			},
			Spec: v1alpha1.OpAMPBridgeSpec{
				Image:    "ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:0.69.0",
				Endpoint: "ws://opamp-server:4320/v1/opamp",
				Headers:  map[string]string{"authorization": "access-12345-token"},
				Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
					v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
					v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
					v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
					v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
					v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
					v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
					v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
					v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
					v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
					v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
					v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
				},
				ComponentsAllowed: map[string][]string{"receivers": {"otlp"}, "processors": {"memory_limiter"}, "exporters": {"logging"}},
			},
		}

		cfg := config.New()

		params := manifests.Params{
			Config:      cfg,
			OpAMPBridge: opampBridge,
			Log:         logger,
		}

		actual, err := ConfigMap(params)
		assert.NoError(t, err)

		assert.Equal(t, "my-instance-opamp-bridge", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)
	})
}
