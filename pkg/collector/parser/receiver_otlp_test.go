package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOTLPSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("otlp"))
}

func TestOTLPIsFoundByName(t *testing.T) {
	// test
	p := For(logger, "otlp", map[interface{}]interface{}{})

	// verify
	assert.Equal(t, "__otlp", p.ParserName())
}

func TestOTLPPortsOverridden(t *testing.T) {
	// prepare
	builder := NewOTLPReceiverParser(logger, "otlp", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{
				"endpoint": "0.0.0.0:1234",
			},
		},
	})

	// test
	ports, err := builder.Ports()

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 1234, ports[0].Port)
}

func TestOTLPExposeDefaultPorts(t *testing.T) {
	// prepare
	builder := NewOTLPReceiverParser(logger, "otlp", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"otlp-grpc":        {portNumber: 4317},
		"otlp-grpc-legacy": {portNumber: 55680},
	}

	// test
	ports, err := builder.Ports()

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 2)

	for _, port := range ports {
		r := expectedResults[port.Name]
		r.seen = true
		expectedResults[port.Name] = r
		assert.EqualValues(t, r.portNumber, port.Port)
	}
	for k, v := range expectedResults {
		assert.True(t, v.seen, "the port %s wasn't included in the service ports", k)
	}
}
