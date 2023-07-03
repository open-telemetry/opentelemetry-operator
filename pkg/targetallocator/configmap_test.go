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

package targetallocator

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLables := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.47.0",
	}

	t.Run("should return expected target allocator config map", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-targetallocator"
		expectedLables["app.kubernetes.io/name"] = "test-targetallocator"

		expectedData := map[string]string{
			"targetallocator.yaml": `allocation_strategy: least-weighted
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
label_selector:
  app.kubernetes.io/component: opentelemetry-collector
  app.kubernetes.io/instance: default.test
  app.kubernetes.io/managed-by: opentelemetry-operator
`,
		}

		actual, err := ConfigMap(collectorInstance())
		assert.NoError(t, err)

		assert.Equal(t, "test-targetallocator", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})
	t.Run("should return expected target allocator config map with label selectors", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-targetallocator"
		expectedLables["app.kubernetes.io/name"] = "test-targetallocator"

		expectedData := map[string]string{
			"targetallocator.yaml": `allocation_strategy: least-weighted
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
label_selector:
  app.kubernetes.io/component: opentelemetry-collector
  app.kubernetes.io/instance: default.test
  app.kubernetes.io/managed-by: opentelemetry-operator
pod_monitor_selector:
  release: test
service_monitor_selector:
  release: test
`,
		}
		instance := collectorInstance()
		instance.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector = map[string]string{
			"release": "test",
		}
		instance.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector = map[string]string{
			"release": "test",
		}
		actual, err := ConfigMap(instance)
		assert.NoError(t, err)

		assert.Equal(t, "test-targetallocator", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

}

func collectorInstance() v1alpha1.OpenTelemetryCollector {
	replicas := int32(2)
	configYAML, err := os.ReadFile("testdata/test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}
	return v1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "opentelemetry.io",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator:0.47.0",
			Ports: []v1.ServicePort{{
				Name: "web",
				Port: 80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 80,
				},
				NodePort: 0,
			}},
			Replicas: &replicas,
			Config:   string(configYAML),
			Mode:     v1alpha1.ModeStatefulSet,
		},
	}
}
