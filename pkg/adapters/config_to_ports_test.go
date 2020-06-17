package adapters

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var logger logr.Logger = logf.Log.WithName("unit-tests")

func TestParsePortFromEndpoint(t *testing.T) {
	for _, tt := range []struct {
		endpoint      string
		expected      int32
		errorExpected bool
	}{
		// prepare
		{"http://localhost:1234", 1234, false},
		{"0.0.0.0:1234", 1234, false},
		{":1234", 1234, false},
		{"no-port", 0, true},
	} {

		// test
		val, err := portFromEndpoint(tt.endpoint)

		// verify
		if tt.errorExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tt.expected, val, "wrong port from endpoint %s: %d", tt.endpoint, val)
	}
}

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
	ports, err := ConfigToReceiverPorts(logger, config)

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
		ports, err := ConfigToReceiverPorts(logger, config)

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
		ports, err := ConfigToReceiverPorts(logger, config)

		// verify
		assert.NoError(t, err)
		assert.NotNil(t, ports)
		assert.Len(t, ports, 0)
	}

}

func TestPortName(t *testing.T) {
	for _, tt := range []struct {
		candidate string
		port      int32
		expected  string
	}{
		{
			candidate: "my-receiver",
			port:      123,
			expected:  "my-receiver",
		},
		{
			candidate: "long-name-long-name-long-name-long-name-long-name-long-name-long-name-long-name",
			port:      123,
			expected:  "port-123",
		},
		{
			candidate: "my-🦄-receiver",
			port:      123,
			expected:  "port-123",
		},
		{
			candidate: "-my-receiver",
			port:      123,
			expected:  "port-123",
		},
	} {
		assert.Equal(t, tt.expected, portName(tt.candidate, tt.port))
	}
}
