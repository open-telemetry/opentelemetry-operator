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

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLables := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.47.0",
	}

	t.Run("should return expected collector config map", func(t *testing.T) {

		expectedData := map[string]string{
			"collector.yaml": `receivers:
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
      exporters: [debug]`,
		}

		param := deploymentParams()
		hash, _ := manifestutils.GetConfigMapSHA(param.OtelCol.Spec.Config)
		expectedName := naming.ConfigMap("test", hash)

		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "0.47.0"

		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, expectedName, actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, len(expectedData), len(actual.Data))
		for k, expected := range expectedData {
			assert.YAMLEq(t, expected, actual.Data[k])
		}
	})

	t.Run("should return expected escaped collector config map with target_allocator config block", func(t *testing.T) {
		expectedData := map[string]string{
			"collector.yaml": `exporters:
  debug:
receivers:
  prometheus:
    config: {}
    target_allocator:
      collector_id: ${POD_NAME}
      endpoint: http://test-targetallocator.default.svc.cluster.local:80
      interval: 30s
service:
  pipelines:
    metrics:
      exporters:
      - debug
      receivers:
      - prometheus
`,
		}

		param, err := newParams("test/test-img", "testdata/http_sd_config_servicemonitor_test.yaml")
		assert.NoError(t, err)

		hash, _ := manifestutils.GetConfigMapSHA(param.OtelCol.Spec.Config)
		expectedName := naming.ConfigMap("test", hash)

		expectedLables["app.kubernetes.io/component"] = "opentelemetry-collector"
		expectedLables["app.kubernetes.io/name"] = "test-collector"
		expectedLables["app.kubernetes.io/version"] = "latest"

		param.OtelCol.Spec.TargetAllocator.Enabled = true
		actual, err := ConfigMap(param)

		assert.NoError(t, err)
		assert.Equal(t, expectedName, actual.Name)
		assert.Equal(t, expectedLables, actual.Labels)
		assert.Equal(t, len(expectedData), len(actual.Data))
		for k, expected := range expectedData {
			assert.YAMLEq(t, expected, actual.Data[k])
		}

		// Reset the value
		expectedLables["app.kubernetes.io/version"] = "0.47.0"
		assert.NoError(t, err)

	})

}
