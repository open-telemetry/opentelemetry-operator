package parser

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger        = logf.Log.WithName("unit-tests")
	ctxWithLogger = context.WithValue(context.Background(), opentelemetry.ContextLogger, logger)
)

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

func TestReceiverType(t *testing.T) {
	for _, tt := range []struct {
		name     string
		expected string
	}{
		{
			name:     "myreceiver",
			expected: "myreceiver",
		},
		{
			name:     "myreceiver/custom",
			expected: "myreceiver",
		},
	} {
		// test
		typ := receiverType(tt.name)

		// assert
		assert.Equal(t, tt.expected, typ)
	}
}

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

func TestEndpointInConfigurationIsntString(t *testing.T) {
	// prepare
	config := map[interface{}]interface{}{
		"endpoint": 123,
	}

	// test
	p := singlePortFromConfigEndpoint(ctxWithLogger, "myreceiver", config)

	// verify
	assert.Nil(t, p)
}

func TestRetrieveGenericParser(t *testing.T) {
	// test
	parser := For("myreceiver", map[interface{}]interface{}{})

	// verify
	assert.Equal(t, parserNameGeneric, parser.ParserName())
}

func TestRetrieveParserFor(t *testing.T) {
	// prepare
	builderCalled := false
	Register("mock", func(name string, config map[interface{}]interface{}) ReceiverParser {
		builderCalled = true
		return &mockParser{}
	})

	// test
	For("mock", map[interface{}]interface{}{})

	// verify
	assert.True(t, builderCalled)
}

type mockParser struct {
	portsFunc func(context.Context) ([]corev1.ServicePort, error)
}

func (m *mockParser) Ports(context.Context) ([]corev1.ServicePort, error) {
	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock"
}
