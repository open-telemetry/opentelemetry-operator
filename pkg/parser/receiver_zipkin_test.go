package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewZipkinParser(t *testing.T) {
	// test
	builder := NewZipkinReceiverParser("zipkin", map[interface{}]interface{}{})

	// verify
	assert.Equal(t, parserNameZipkin, builder.ParserName())
}

func TestZipkinParserDefaultPort(t *testing.T) {
	// prepare
	builder := NewZipkinReceiverParser("zipkin", map[interface{}]interface{}{})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 9411, ports[0].Port)
	assert.Equal(t, "zipkin", ports[0].Name)
}

func TestZipkinParserOverridePort(t *testing.T) {
	// prepare
	builder := NewZipkinReceiverParser("zipkin", map[interface{}]interface{}{
		"endpoint": "0.0.0.0:9412",
	})

	// test
	ports, err := builder.Ports(ctxWithLogger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 9412, ports[0].Port)
}
