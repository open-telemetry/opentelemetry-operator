package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenericReceiverWithEndpoint(t *testing.T) {
	builder := NewGenericReceiverParser("myreceiver", map[interface{}]interface{}{
		"endpoint": "0.0.0.0:1234",
	})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.Equal(t, int32(1234), ports[0].Port)
}

func TestGenericReceiverWithInvalidEndpoint(t *testing.T) {
	// there's no parser regitered to handle "myreceiver", so, it falls back to the generic parser
	builder := NewGenericReceiverParser("myreceiver", map[interface{}]interface{}{
		"endpoint": "0.0.0.0",
	})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 0)
}
