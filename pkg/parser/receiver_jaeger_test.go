package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJaegerParserRegistration(t *testing.T) {
	// verify
	assert.Contains(t, registry, "jaeger")
}

func TestJaegerMinimalReceiverConfiguration(t *testing.T) {
	builder := NewJaegerReceiverParser("jaeger", map[interface{}]interface{}{
		"grpc": map[interface{}]interface{}{},
	})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.Equal(t, int32(defaultGRPCPort), ports[0].Port)
}

func TestJaegerOverrideReceiverProtocolPort(t *testing.T) {
	builder := NewJaegerReceiverParser("jaeger", map[interface{}]interface{}{
		"grpc": map[interface{}]interface{}{
			"endpoint": "0.0.0.0:1234",
		},
	})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.Equal(t, int32(1234), ports[0].Port)
}

func TestJaegerDefaultPorts(t *testing.T) {
	builder := NewJaegerReceiverParser("jaeger", map[interface{}]interface{}{
		"grpc":           map[interface{}]interface{}{},
		"thrift_http":    map[interface{}]interface{}{},
		"thrift_compact": map[interface{}]interface{}{},
		"thrift_binary":  map[interface{}]interface{}{},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"jaeger-grpc":           {portNumber: defaultGRPCPort},
		"jaeger-thrift-http":    {portNumber: defaultThriftHTTPPort},
		"jaeger-thrift-compact": {portNumber: defaultThriftCompactPort},
		"jaeger-thrift-binary":  {portNumber: defaultThriftBinaryPort},
	}

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 4)

	for _, port := range ports {
		r := expectedResults[port.Name]
		r.seen = true
		expectedResults[port.Name] = r
	}

	for k, v := range expectedResults {
		assert.True(t, v.seen, "the port %s wasn't included in the service ports", k)
	}
}

func TestJaegerParserName(t *testing.T) {
	// test
	p := For("jaeger", map[interface{}]interface{}{})

	// verify
	assert.Equal(t, parserNameJaeger, p.ParserName())
}
