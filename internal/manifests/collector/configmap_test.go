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
	"github.com/stretchr/testify/require"
	colfg "go.opentelemetry.io/collector/featuregate"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
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
      endpoint: http://test-targetallocator:80
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

	t.Run("should return expected escaped collector config map with target_allocator and https config block", func(t *testing.T) {
		expectedData := map[string]string{
			"collector.yaml": `exporters:
  debug:
receivers:
  prometheus:
    config: {}
    target_allocator:
      collector_id: ${POD_NAME}
      endpoint: https://test-targetallocator:443
      interval: 30s
      tls:
        ca_file: /tls/ca.crt
        cert_file: /tls/tls.crt
        key_file: /tls/tls.key
service:
  pipelines:
    metrics:
      exporters:
      - debug
      receivers:
      - prometheus
`,
		}

		param, err := newParams("test/test-img", "testdata/http_sd_config_servicemonitor_test.yaml", config.WithCertManagerAvailability(certmanager.Available))
		require.NoError(t, err)
		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err = flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

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
