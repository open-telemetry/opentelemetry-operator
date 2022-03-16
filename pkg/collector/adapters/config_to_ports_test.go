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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/parser"
)

var logger = logf.Log.WithName("unit-tests")

func TestExtractPortsFromConfig(t *testing.T) {
	configStr := `receivers:
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
      exporters: [logging]
    metrics/1:
      receivers: [jaeger, jaeger/custom]
      exporters: [logging]
    metrics/1:
      receivers: [otlp, otlp/2, zipkin]
      exporters: [logging]
`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	ports, err := adapters.ConfigToReceiverPorts(logger, config)
	assert.NoError(t, err)
	assert.Len(t, ports, 12)

	// verify
	expectedPorts := map[int32]bool{}
	expectedPorts[int32(12345)] = false
	expectedPorts[int32(12346)] = false
	expectedPorts[int32(14250)] = false
	expectedPorts[int32(6831)] = false
	expectedPorts[int32(6833)] = false
	expectedPorts[int32(15268)] = false
	expectedPorts[int32(4318)] = false
	expectedPorts[int32(55681)] = false
	expectedPorts[int32(55555)] = false
	expectedPorts[int32(9411)] = false
	expectedPorts[int32(33333)] = false

	expectedNames := map[string]bool{}
	expectedNames["examplereceiver"] = false
	expectedNames["examplereceiver-settings"] = false
	expectedNames["jaeger-grpc"] = false
	expectedNames["jaeger-thrift-compact"] = false
	expectedNames["jaeger-thrift-binary"] = false
	expectedNames["jaeger-custom-thrift-http"] = false
	expectedNames["otlp-grpc"] = false
	expectedNames["otlp-http"] = false
	expectedNames["otlp-http-legacy"] = false
	expectedNames["otlp-2-grpc"] = false
	expectedNames["zipkin"] = false
	expectedNames["zipkin-2"] = false

	expectedAppProtocols := map[string]string{}
	expectedAppProtocols["otlp-grpc"] = "grpc"
	expectedAppProtocols["otlp-http"] = "http"
	expectedAppProtocols["otlp-http-legacy"] = "http"
	expectedAppProtocols["jaeger-custom-thrift-http"] = "http"
	expectedAppProtocols["jaeger-grpc"] = "grpc"
	expectedAppProtocols["otlp-2-grpc"] = "grpc"
	expectedAppProtocols["zipkin"] = "http"
	expectedAppProtocols["zipkin-2"] = "http"

	// make sure we only have the ports in the set
	for _, port := range ports {
		assert.NotNil(t, expectedPorts[port.Port])
		assert.NotNil(t, expectedNames[port.Name])
		expectedPorts[port.Port] = true
		expectedNames[port.Name] = true

		if appProtocol, ok := expectedAppProtocols[port.Name]; ok {
			assert.Equal(t, appProtocol, *port.AppProtocol)
		}
	}

	// and make sure all the ports from the set are there
	for _, val := range expectedPorts {
		assert.True(t, val)
	}

	// make sure we only have the ports names in the set
	for _, val := range expectedNames {
		assert.True(t, val)
	}
}

func TestNoPortsParsed(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		configStr string
		expected  error
	}{
		{
			"empty",
			"",
			adapters.ErrNoReceivers,
		},
		{
			"not a map",
			"receivers: some-string",
			adapters.ErrReceiversNotAMap,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			config, err := adapters.ConfigFromString(tt.configStr)
			require.NoError(t, err)

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)

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
			"receivers:\n  some-receiver: string",
		},
		{
			"receiver's endpoint isn't string",
			"receivers:\n  some-receiver:\n    endpoint: 123",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			config, err := adapters.ConfigFromString(tt.configStr)
			require.NoError(t, err)

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)

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
	parser.Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ReceiverParser {
		return mockParser
	})

	config := map[interface{}]interface{}{
		"receivers": map[interface{}]interface{}{
			"mock": map[interface{}]interface{}{},
		},
	}

	// test
	ports, err := adapters.ConfigToReceiverPorts(logger, config)

	// verify
	assert.Len(t, ports, 0)
	assert.NoError(t, err)
	assert.True(t, mockParserCalled)
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
