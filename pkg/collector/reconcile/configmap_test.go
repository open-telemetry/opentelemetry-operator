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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
)

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create collector and target allocator config maps", func(t *testing.T) {
		param := params()
		configMap, err := targetallocator.ConfigMap(param.Config, param.Log, param.Instance)
		assert.NoError(t, err)

		err = expectedConfigMaps(context.Background(), params(), []*v1.ConfigMap{collector.ConfigMap(param.Config, param.Log, param.Instance), configMap}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update collector config map", func(t *testing.T) {

		param := manifests.Params{
			Config: config.New(),
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
			},
			Log: logger,
		}
		cm := collector.ConfigMap(param.Config, param.Log, param.Instance)
		createObjectIfNotExists(t, "test-collector", cm)

		param = params()

		err := expectedConfigMaps(context.Background(), params(), []*v1.ConfigMap{collector.ConfigMap(param.Config, param.Log, param.Instance)}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, params().Instance.Spec.Config, actual.Data["collector.yaml"])
	})

	t.Run("should update target allocator config map", func(t *testing.T) {

		param := manifests.Params{
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
		}
		cm, err := targetallocator.ConfigMap(param.Config, param.Log, param.Instance)
		assert.EqualError(t, err, "no receivers available as part of the configuration")
		createObjectIfNotExists(t, "test-targetallocator", cm)

		newParam := params()
		configMap, err := targetallocator.ConfigMap(newParam.Config, newParam.Log, newParam.Instance)
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), params(), []*v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

		promConfig, err := ta.ConfigToPromConfig(params().Instance.Spec.Config)
		assert.NoError(t, err)

		taConfig := make(map[interface{}]interface{})
		taConfig["label_selector"] = map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-collector",
			"app.kubernetes.io/part-of":    "opentelemetry",
		}
		taConfig["config"] = promConfig["config"]
		taConfig["allocation_strategy"] = "least-weighted"
		taConfigYAML, _ := yaml.Marshal(taConfig)

		assert.Equal(t, string(taConfigYAML), actual.Data["targetallocator.yaml"])
	})

	t.Run("should delete config map", func(t *testing.T) {

		deletecm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-collector",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-collector", &deletecm)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-collector"})
		assert.True(t, exists)
		param := params()
		err := deleteConfigMaps(context.Background(), params(), []*v1.ConfigMap{collector.ConfigMap(param.Config, param.Log, param.Instance)})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-collector"})
		assert.False(t, exists)
	})
}
