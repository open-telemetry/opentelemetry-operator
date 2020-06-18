package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger        = logf.Log.WithName("unit-tests")
	ctxWithLogger = context.WithValue(context.Background(), opentelemetry.ContextLogger, logger)
)

func TestExtractPortsFromConfig(t *testing.T) {
	// prepare
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
    grpc:
    thrift_compact:
    thrift_binary:
      endpoint: 0.0.0.0:6833
  jaeger/custom:
    thrift_http:
      endpoint: 0.0.0.0:15268
`

	// test
	config, err := ConfigFromString(configStr)
	require.NoError(t, err)
	ports, err := ConfigToReceiverPorts(ctxWithLogger, config)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 6)

	expectedPorts := map[int32]bool{}
	expectedPorts[int32(12345)] = false
	expectedPorts[int32(12346)] = false
	expectedPorts[int32(14250)] = false
	expectedPorts[int32(6831)] = false
	expectedPorts[int32(6833)] = false
	expectedPorts[int32(15268)] = false

	expectedNames := map[string]bool{}
	expectedNames["examplereceiver"] = false
	expectedNames["examplereceiver-settings"] = false
	expectedNames["jaeger-grpc"] = false
	expectedNames["jaeger-thrift-compact"] = false
	expectedNames["jaeger-thrift-binary"] = false
	expectedNames["jaeger-custom-thrift-http"] = false

	// make sure we only have the ports in the set
	for _, port := range ports {
		assert.Contains(t, expectedPorts, port.Port)
		assert.Contains(t, expectedNames, port.Name)
		expectedPorts[port.Port] = true
		expectedNames[port.Name] = true
	}

	// and make sure all the ports from the set are there
	for _, val := range expectedPorts {
		assert.True(t, val)
	}
}

func TestNoPortsParsed(t *testing.T) {
	for _, tt := range []struct {
		config   string
		expected error
	}{
		// prepare
		{
			config:   "",
			expected: ErrNoReceivers,
		},
		{
			config:   "receivers: some-string",
			expected: ErrReceiversNotAMap,
		},
	} {
		config, err := ConfigFromString(tt.config)
		require.NoError(t, err)

		// test
		ports, err := ConfigToReceiverPorts(ctxWithLogger, config)

		// verify
		assert.Equal(t, tt.expected, err)
		assert.Nil(t, ports)
	}
}

func TestInvalidReceiver(t *testing.T) {
	for _, tt := range []string{
		// the receiver isn't a map, can't have an endpoint
		"receivers:\n  some-receiver: string",

		// the receiver's endpoint isn't a string, can't parse the port from it
		"receivers:\n  some-receiver:\n    endpoint: 123",
	} {
		// test
		config, err := ConfigFromString(tt)
		require.NoError(t, err)
		ports, err := ConfigToReceiverPorts(ctxWithLogger, config)

		// verify
		assert.NoError(t, err)
		assert.NotNil(t, ports)
		assert.Len(t, ports, 0)
	}

}

func TestParserFailed(t *testing.T) {
	// prepare
	mockParserCalled := false
	mockParser := &mockParser{
		portsFunc: func(context.Context) ([]v1.ServicePort, error) {
			mockParserCalled = true
			return nil, errors.New("mocked error")
		},
	}
	parser.Register("mock", func(name string, config map[interface{}]interface{}) parser.ReceiverParser {
		return mockParser
	})

	config := map[interface{}]interface{}{
		"receivers": map[interface{}]interface{}{
			"mock": map[interface{}]interface{}{},
		},
	}

	// test
	ports, err := ConfigToReceiverPorts(ctxWithLogger, config)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 0)
	assert.True(t, mockParserCalled)
}

type mockParser struct {
	portsFunc func(context.Context) ([]corev1.ServicePort, error)
}

func (m *mockParser) Ports(ctx context.Context) ([]corev1.ServicePort, error) {
	if m.portsFunc != nil {
		return m.portsFunc(ctx)
	}

	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock-adapters"
}
