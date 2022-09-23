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
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLables := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.47.0",
	}

	t.Run("should return expected collector config map", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "0.47.0"

		expectedData := map[string]string{
			"collector.yaml": `processors:
receivers:
  jaeger:
    protocols:
      grpc:
  prometheus:
    config:
      scrape_configs:
      - job_name: otel-collector
        scrape_interval: 10s
        static_configs:
          - targets: [ '0.0.0.0:8888', '0.0.0.0:9999' ]

exporters:
  logging:

service:
  pipelines:
    metrics:
      receivers: [prometheus, jaeger]
      processors: []
      exporters: [logging]`,
		}

		actual := desiredConfigMap(context.Background(), params())

		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

	t.Run("should return expected collector config map with http_sd_config", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"

		expectedData := map[string]string{
			"collector.yaml": `exporters:
  logging: null
processors: null
receivers:
  jaeger:
    protocols:
      grpc: null
  prometheus:
    config:
      global:
        scrape_interval: 1m
        scrape_timeout: 10s
        evaluation_interval: 1m
      scrape_configs:
      - job_name: otel-collector
        honor_timestamps: true
        scrape_interval: 10s
        scrape_timeout: 10s
        metrics_path: /metrics
        scheme: http
        follow_redirects: true
        http_sd_configs:
        - follow_redirects: false
          url: http://test-targetallocator:80/jobs/otel-collector/targets?collector_id=$POD_NAME
service:
  pipelines:
    metrics:
      exporters:
      - logging
      processors: []
      receivers:
      - prometheus
      - jaeger
`,
		}

		param := params()
		param.Instance.Spec.TargetAllocator.Enabled = true
		actual := desiredConfigMap(context.Background(), param)

		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

	t.Run("should return expected escaped collector config map with http_sd_config", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "latest"

		expectedData := map[string]string{
			"collector.yaml": `exporters:
  logging: null
processors: null
receivers:
  prometheus:
    config:
      global:
        scrape_interval: 1m
        scrape_timeout: 10s
        evaluation_interval: 1m
      scrape_configs:
      - job_name: serviceMonitor/test/test/0
        honor_timestamps: true
        scrape_interval: 1m
        scrape_timeout: 10s
        metrics_path: /metrics
        scheme: http
        follow_redirects: true
        http_sd_configs:
        - follow_redirects: false
          url: http://test-targetallocator:80/jobs/serviceMonitor%2Ftest%2Ftest%2F0/targets?collector_id=$POD_NAME
service:
  pipelines:
    metrics:
      exporters:
      - logging
      processors: []
      receivers:
      - prometheus
`,
		}

		param, err := newParams("test/test-img", "../testdata/http_sd_config_servicemonitor_test.yaml")
		assert.NoError(t, err)
		param.Instance.Spec.TargetAllocator.Enabled = true
		actual := desiredConfigMap(context.Background(), param)

		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

		// Reset the value
		expectedLables["app.kubernetes.io/version"] = "0.47.0"

	})

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
filter_strategy: no-op
label_selector:
  app.kubernetes.io/component: opentelemetry-collector
  app.kubernetes.io/instance: default.test
  app.kubernetes.io/managed-by: opentelemetry-operator
`,
		}

		actual, err := desiredTAConfigMap(params())
		assert.NoError(t, err)

		assert.Equal(t, "test-targetallocator", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

}

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create collector and target allocator config maps", func(t *testing.T) {
		configMap, err := desiredTAConfigMap(params())
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{desiredConfigMap(context.Background(), params()), configMap}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update collector config map", func(t *testing.T) {

		param := Params{
			Config: config.New(),
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
			},
			Scheme:   testScheme,
			Log:      logger,
			Recorder: record.NewFakeRecorder(10),
		}
		cm := desiredConfigMap(context.Background(), param)
		createObjectIfNotExists(t, "test-collector", &cm)

		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{desiredConfigMap(context.Background(), params())}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, params().Instance.Spec.Config, actual.Data["collector.yaml"])
	})

	t.Run("should update target allocator config map", func(t *testing.T) {

		param := Params{
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
		cm, err := desiredTAConfigMap(param)
		assert.EqualError(t, err, "no receivers available as part of the configuration")
		createObjectIfNotExists(t, "test-targetallocator", &cm)

		configMap, err := desiredTAConfigMap(params())
		assert.NoError(t, err)
		err = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

		parmConfig, err := ta.ConfigToPromConfig(params().Instance.Spec.Config)
		assert.NoError(t, err)

		taConfig := make(map[interface{}]interface{})
		taConfig["label_selector"] = map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-collector",
		}
		taConfig["config"] = parmConfig
		taConfig["allocation_strategy"] = "least-weighted"
		taConfig["filter_strategy"] = "no-op"
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

		err := deleteConfigMaps(context.Background(), params(), []v1.ConfigMap{desiredConfigMap(context.Background(), params())})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-collector"})
		assert.False(t, exists)
	})
}
