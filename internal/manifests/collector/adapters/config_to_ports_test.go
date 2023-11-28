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

package adapters_test

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/receiver"
)

var logger = logf.Log.WithName("unit-tests")

var portConfigStr = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  examplereceiver/invalid-ignored:
    endpoint: "0.0.0.0"
  examplereceiver/invalid-not-number:
    endpoint: "0.0.0.0:not-number"
  examplereceiver/without-endpoint:
    notendpoint: "0.0.0.0:12347"
  jaeger:
    protocols:
      grpc:
      thrift_compact:
      thrift_binary:
        endpoint: 0.0.0.0:6833
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
  otlp:
    protocols:
      grpc:
      http:
  otlp/2:
    protocols:
      grpc:
        endpoint: 0.0.0.0:55555
  zipkin:
  zipkin/2:
    endpoint: 0.0.0.0:33333
service:
  pipelines:
    metrics:
      receivers: [examplereceiver, examplereceiver/settings]
      exporters: [debug]
    metrics/1:
      receivers: [jaeger, jaeger/custom]
      exporters: [debug]
    metrics/2:
      receivers: [otlp, otlp/2, zipkin]
      exporters: [debug]
`

func TestExtractPortsFromConfig(t *testing.T) {
	// prepare
	config, err := adapters.ConfigFromString(portConfigStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	ports, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeReceiver, config)
	assert.NoError(t, err)
	assert.Len(t, ports, 10)

	// verify
	httpAppProtocol := "http"
	grpcAppProtocol := "grpc"
	targetPortZero := intstr.IntOrString{Type: 0, IntVal: 0, StrVal: ""}
	targetPort4317 := intstr.IntOrString{Type: 0, IntVal: 4317, StrVal: ""}
	targetPort4318 := intstr.IntOrString{Type: 0, IntVal: 4318, StrVal: ""}

	expectedPorts := []corev1.ServicePort{
		{Name: "examplereceiver", Port: 12345},
		{Name: "port-12346", Port: 12346},
		{Name: "port-15268", AppProtocol: &httpAppProtocol, Protocol: "TCP", Port: 15268, TargetPort: targetPortZero},
		{Name: "jaeger-grpc", AppProtocol: &grpcAppProtocol, Protocol: "TCP", Port: 14250},
		{Name: "port-6833", Protocol: "UDP", Port: 6833},
		{Name: "port-6831", Protocol: "UDP", Port: 6831},
		{Name: "otlp-2-grpc", AppProtocol: &grpcAppProtocol, Protocol: "TCP", Port: 55555},
		{Name: "otlp-grpc", AppProtocol: &grpcAppProtocol, Port: 4317, TargetPort: targetPort4317},
		{Name: "otlp-http", AppProtocol: &httpAppProtocol, Port: 4318, TargetPort: targetPort4318},
		{Name: "zipkin", AppProtocol: &httpAppProtocol, Protocol: "TCP", Port: 9411},
	}
	assert.ElementsMatch(t, expectedPorts, ports)
}

func TestNoPortsParsed(t *testing.T) {
	for _, tt := range []struct {
		expected  error
		desc      string
		configStr string
	}{
		{
			expected:  errors.New("no receivers available as part of the configuration"),
			desc:      "empty",
			configStr: "",
		},
		{
			expected:  errors.New("receivers doesn't contain valid components"),
			desc:      "not a map",
			configStr: "receivers: some-string",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			config, err := adapters.ConfigFromString(tt.configStr)
			require.NoError(t, err)

			// test
			ports, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeReceiver, config)

			// verify
			assert.Nil(t, ports)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestInvalidReceivers(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		configStr string
	}{
		{
			"receiver isn't a map",
			"receivers:\n  some-receiver: string\nservice:\n  pipelines:\n    metrics:\n      receivers: [some-receiver]",
		},
		{
			"receiver's endpoint isn't string",
			"receivers:\n  some-receiver:\n    endpoint: 123\nservice:\n  pipelines:\n    metrics:\n      receivers: [some-receiver]",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			config, err := adapters.ConfigFromString(tt.configStr)
			require.NoError(t, err)

			// test
			ports, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeReceiver, config)

			// verify
			assert.NoError(t, err)
			assert.Len(t, ports, 0)
		})
	}
}

func TestParserFailed(t *testing.T) {
	// prepare
	mockParserCalled := false
	mockParser := &mockParser{
		portsFunc: func() ([]corev1.ServicePort, error) {
			mockParserCalled = true
			return nil, errors.New("mocked error")
		},
	}
	receiver.Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ComponentPortParser {
		return mockParser
	})

	config := map[interface{}]interface{}{
		"receivers": map[interface{}]interface{}{
			"mock": map[string]interface{}{},
		},
		"service": map[interface{}]interface{}{
			"pipelines": map[interface{}]interface{}{
				"metrics": map[interface{}]interface{}{
					"receivers": []interface{}{"mock"},
				},
			},
		},
	}

	// test
	ports, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeReceiver, config)

	// verify
	assert.Len(t, ports, 0)
	assert.NoError(t, err)
	assert.True(t, mockParserCalled)
}

func TestConfigToMetricsPort(t *testing.T) {
	t.Run("custom port specified", func(t *testing.T) {
		config := map[interface{}]interface{}{
			"service": map[interface{}]interface{}{
				"telemetry": map[interface{}]interface{}{
					"metrics": map[interface{}]interface{}{
						"address": "0.0.0.0:9090",
					},
				},
			},
		}

		port, err := adapters.ConfigToMetricsPort(logger, config)
		assert.NoError(t, err)
		assert.Equal(t, int32(9090), port)
	})

	for _, tt := range []struct {
		desc   string
		config map[interface{}]interface{}
	}{
		{
			"bad address",
			map[interface{}]interface{}{
				"service": map[interface{}]interface{}{
					"telemetry": map[interface{}]interface{}{
						"metrics": map[interface{}]interface{}{
							"address": "0.0.0.0",
						},
					},
				},
			},
		},
		{
			"missing address",
			map[interface{}]interface{}{
				"service": map[interface{}]interface{}{
					"telemetry": map[interface{}]interface{}{
						"metrics": map[interface{}]interface{}{
							"level": "detailed",
						},
					},
				},
			},
		},
		{
			"missing metrics",
			map[interface{}]interface{}{
				"service": map[interface{}]interface{}{
					"telemetry": map[interface{}]interface{}{},
				},
			},
		},
		{
			"missing telemetry",
			map[interface{}]interface{}{
				"service": map[interface{}]interface{}{},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// these are acceptable failures, we return to the collector's default metric port
			port, err := adapters.ConfigToMetricsPort(logger, tt.config)
			assert.NoError(t, err)
			assert.Equal(t, int32(8888), port)
		})
	}
}

type mockParser struct {
	portsFunc func() ([]corev1.ServicePort, error)
}

func (m *mockParser) Ports() ([]corev1.ServicePort, error) {
	if m.portsFunc != nil {
		return m.portsFunc()
	}

	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock-adapters"
}
