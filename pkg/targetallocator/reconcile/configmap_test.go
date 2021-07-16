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

package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expectedLables := map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/instance":   "test.targetallocator",
			"app.kubernetes.io/part-of":    "opentelemetry",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
			"app.kubernetes.io/name":       "test-targetallocator",
		}

		expectedData := map[string]string{
			"targetallocator.yaml": `config:
  scrape_configs:
    job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
label_selector:
  app.kubernetes.io/instance: default.test
  app.kubernetes.io/managed-by: opentelemetry-operator
`,
		}

		actual, err := desiredConfigMap(context.Background(), params())
		assert.NoError(t, err)

		assert.Equal(t, "test-targetallocator", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

}

func TestExpectedConfigMap(t *testing.T) {
	param := params()
	t.Run("should create config map", func(t *testing.T) {
		configMap, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), param, []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update config map", func(t *testing.T) {

		newParam := Params{
			Client: k8sClient,
			Instance: v1alpha1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "opentelemetry.io",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					UID:       instanceUID,
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeStatefulSet,
					Ports: []v1.ServicePort{{
						Name: "web",
						Port: 80,
						TargetPort: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 80,
						},
						NodePort: 0,
					}},
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled: true,
					},
					Config: "",
				},
			},
			Scheme: testScheme,
			Log:    logger,
		}
		cm, err := desiredConfigMap(context.Background(), newParam)
		assert.EqualError(t, err, "no receivers available as part of the configuration")
		createObjectIfNotExists(t, "test-targetallocator", &cm)

		configMap, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), param, []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

		config, err := adapters.ConfigFromString(param.Instance.Spec.Config)
		assert.NoError(t, err)

		parmConfig, err := ta.ConfigToPromConfig(config)
		assert.NoError(t, err)

		taConfig := make(map[interface{}]interface{})
		taConfig["label_selector"] = map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}
		taConfig["config"] = parmConfig
		taConfigYAML, _ := yaml.Marshal(taConfig)

		assert.Equal(t, string(taConfigYAML), actual.Data["targetallocator.yaml"])
	})

	t.Run("should delete config map", func(t *testing.T) {

		deletecm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-targetallocator",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "test.targetallocator",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-targetallocator", &deletecm)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.True(t, exists)

		configMap, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		err = deleteConfigMaps(context.Background(), param, []v1.ConfigMap{configMap})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.False(t, exists)
	})
}
