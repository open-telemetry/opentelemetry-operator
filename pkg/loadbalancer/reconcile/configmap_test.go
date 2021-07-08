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

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	lbadapters "github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/adapters"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expectedLables := map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/instance":   "default.loadbalancer",
			"app.kubernetes.io/part-of":    "opentelemetry",
			"app.kubernetes.io/component":  "opentelemetry-loadbalancer",
			"app.kubernetes.io/name":       "test-loadbalancer",
		}

		expectedData := map[string]string{
			"loadbalancer.yaml": `config:
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
mode: LeastConnection
`,
		}

		actual, notify, err := desiredConfigMap(context.Background(), params())
		assert.NoError(t, err)
		assert.Equal(t, notify, "")

		assert.Equal(t, "test-loadbalancer", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

}

func TestExpectedConfigMap(t *testing.T) {
	param := params()
	t.Run("should create config map", func(t *testing.T) {
		configMap, notify, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		assert.Equal(t, notify, "")
		err = expectedConfigMaps(context.Background(), param, []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-loadbalancer"})

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
					LoadBalancer: v1alpha1.OpenTelemetryLoadBalancer{
						Mode: "LeastConnection",
					},
					Config: "",
				},
			},
			Scheme: testScheme,
			Log:    logger,
		}
		cm, notify, err := desiredConfigMap(context.Background(), newParam)
		assert.NoError(t, err)
		assert.Equal(t, notify, "no receivers available as part of the configuration")
		createObjectIfNotExists(t, "test-loadbalancer", &cm)

		configMap, notify, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		assert.Equal(t, notify, "")
		err = expectedConfigMaps(context.Background(), param, []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-loadbalancer"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

		config, err := adapters.ConfigFromString(param.Instance.Spec.Config)
		assert.NoError(t, err)

		parmConfig, notify := lbadapters.ConfigToPromConfig(config)
		assert.Equal(t, notify, "")

		lbConfig := make(map[interface{}]interface{})
		lbConfig["mode"] = "LeastConnection"
		lbConfig["label_selector"] = map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}
		lbConfig["config"] = parmConfig
		lbConfigYAML, _ := yaml.Marshal(lbConfig)

		assert.Equal(t, string(lbConfigYAML), actual.Data["loadbalancer.yaml"])
	})

	t.Run("should delete config map", func(t *testing.T) {

		deletecm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-loadbalancer",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.loadbalancer",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-loadbalancer", &deletecm)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-loadbalancer"})
		assert.True(t, exists)

		configMap, notify, err := desiredConfigMap(context.Background(), param)
		assert.NoError(t, err)
		assert.Equal(t, notify, "")
		err = deleteConfigMaps(context.Background(), param, []v1.ConfigMap{configMap})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-loadbalancer"})
		assert.False(t, exists)
	})
}
