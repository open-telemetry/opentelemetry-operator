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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLables := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.69.0",
	}

	t.Run("should return expected opamp-bridge config map", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-opamp-bridge"
		expectedLables["app.kubernetes.io/name"] = "test-opamp-bridge"

		expectedData := map[string]string{
			"remoteconfiguration.yaml": `capabilities:
- AcceptsRemoteConfig
- ReportsEffectiveConfig
- ReportsOwnTraces
- ReportsOwnMetrics
- ReportsOwnLogs
- AcceptsOpAMPConnectionSettings
- AcceptsOtherConnectionSettings
- AcceptsRestartCommand
- ReportsHealth
- ReportsRemoteConfig
components_allowed:
  exporters:
  - logging
  processors:
  - memory_limiter
  receivers:
  - otlp
endpoint: ws://127.0.0.1:4320/v1/opamp
protocol: wss
`}
		actual, err := desiredConfigMap(context.Background(), params())
		assert.NoError(t, err)

		assert.Equal(t, "test-opamp-bridge", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)
	})
}

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create opamp-bridge and its config map", func(t *testing.T) {
		configMap, err := desiredConfigMap(context.Background(), params())
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update opamp-bridge config map", func(t *testing.T) {

		param := Params{
			Client: k8sClient,
			Instance: v1alpha1.OpAMPBridge{
				TypeMeta: metav1.TypeMeta{
					Kind:       "opentelemetry.io",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					UID:       instanceUID,
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint:     "ws://127.0.0.1:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []v1alpha1.OpAMPBridgeCapability{v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig, v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig},
				},
			},
			Scheme: testScheme,
			Log:    logger,
		}

		cm, err := desiredConfigMap(ctx, param)
		assert.NoError(t, err)
		createObjectIfNotExists(t, "test-opamp-bridge", &cm)

		configMap, err := desiredConfigMap(ctx, params())
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

		opampBridgeConfig := make(map[interface{}]interface{})
		opampBridgeConfig["endpoint"] = "ws://127.0.0.1:4320/v1/opamp"
		opampBridgeConfig["protocol"] = "wss"
		opampBridgeConfig["capabilities"] = []string{"AcceptsRemoteConfig", "ReportsEffectiveConfig", "ReportsOwnTraces", "ReportsOwnMetrics", "ReportsOwnLogs", "AcceptsOpAMPConnectionSettings", "AcceptsOtherConnectionSettings", "AcceptsRestartCommand", "ReportsHealth", "ReportsRemoteConfig"}
		opampBridgeConfig["components_allowed"] = map[string][]string{"receivers": {"otlp"}, "processors": {"memory_limiter"}, "exporters": {"logging"}}

		opampBridgeConfigYAML, _ := yaml.Marshal(opampBridgeConfig)

		assert.Equal(t, string(opampBridgeConfigYAML), actual.Data["remoteconfiguration.yaml"])
	})

	t.Run("should delete config map", func(t *testing.T) {

		deletecm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-opamp-bridge",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-opamp-bridge", &deletecm)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.True(t, exists)

		desiredConfigMap, err := desiredConfigMap(context.Background(), params())
		assert.NoError(t, err)
		err = deleteConfigMaps(context.Background(), params(), []v1.ConfigMap{desiredConfigMap})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.False(t, exists)
	})
}
