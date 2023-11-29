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

package collector

import (
	"testing"

	colfeaturegate "go.opentelemetry.io/collector/featuregate"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
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
  debug:

service:
  pipelines:
    metrics:
      receivers: [prometheus, jaeger]
      processors: []
      exporters: [debug]`,
		}

		param := deploymentParams()
		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

	t.Run("should return expected collector config map with http_sd_config if rewrite flag disabled", func(t *testing.T) {
		err := colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), false)
		assert.NoError(t, err)
		t.Cleanup(func() {
			_ = colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), true)
		})
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"

		expectedData := map[string]string{
			"collector.yaml": `exporters:
  debug: null
processors: null
receivers:
  jaeger:
    protocols:
      grpc: null
  prometheus:
    config:
      scrape_configs:
      - http_sd_configs:
        - url: http://test-targetallocator:80/jobs/otel-collector/targets?collector_id=$POD_NAME
        job_name: otel-collector
        scrape_interval: 10s
service:
  pipelines:
    metrics:
      exporters:
      - debug
      processors: []
      receivers:
      - prometheus
      - jaeger
`,
		}

		param := deploymentParams()
		param.OtelCol.Spec.TargetAllocator.Enabled = true
		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, "test-collector", actual.GetName())
		assert.Equal(t, expectedLables, actual.GetLabels())
		assert.Equal(t, expectedData, actual.Data)

	})

	t.Run("should return expected escaped collector config map with http_sd_config if rewrite flag disabled", func(t *testing.T) {
		err := colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), false)
		assert.NoError(t, err)
		t.Cleanup(func() {
			_ = colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), true)
		})

		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "latest"

		expectedData := map[string]string{
			"collector.yaml": `exporters:
  debug: null
processors: null
receivers:
  prometheus:
    config:
      scrape_configs:
      - http_sd_configs:
        - url: http://test-targetallocator:80/jobs/serviceMonitor%2Ftest%2Ftest%2F0/targets?collector_id=$POD_NAME
        job_name: serviceMonitor/test/test/0
    target_allocator:
      collector_id: ${POD_NAME}
      endpoint: http://test-targetallocator:80
      http_sd_config:
        refresh_interval: 60s
      interval: 30s
service:
  pipelines:
    metrics:
      exporters:
      - debug
      processors: []
      receivers:
      - prometheus
`,
		}

		param, err := newParams("test/test-img", "testdata/http_sd_config_servicemonitor_test_ta_set.yaml")
		assert.NoError(t, err)
		param.OtelCol.Spec.TargetAllocator.Enabled = true
		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

		// Reset the value
		expectedLables["app.kubernetes.io/version"] = "0.47.0"

	})

	t.Run("should return expected escaped collector config map with target_allocator config block", func(t *testing.T) {
		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "latest"

		expectedData := map[string]string{
			"collector.yaml": `exporters:
  debug: null
processors: null
receivers:
  prometheus:
    config: {}
    target_allocator:
      collector_id: ${POD_NAME}
      endpoint: http://test-targetallocator:80
      interval: 30s
service:
  pipelines:
    metrics:
      exporters:
      - debug
      processors: []
      receivers:
      - prometheus
`,
		}

		param, err := newParams("test/test-img", "testdata/http_sd_config_servicemonitor_test.yaml")
		assert.NoError(t, err)
		param.OtelCol.Spec.TargetAllocator.Enabled = true
		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, "test-collector", actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

		// Reset the value
		expectedLables["app.kubernetes.io/version"] = "0.47.0"
		assert.NoError(t, err)

	})

}
