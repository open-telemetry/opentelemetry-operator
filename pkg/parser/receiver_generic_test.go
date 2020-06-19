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

func TestDownstreamParsers(t *testing.T) {
	for _, tt := range []struct {
		receiverName      string
		parserName        string
		parserDefaultPort int32
		builder           func(string, map[interface{}]interface{}) ReceiverParser
	}{
		{"zipkin", parserNameZipkin, 9411, NewZipkinReceiverParser},
		{"opencensus", parserNameOpenCensus, 55678, NewOpenCensusReceiverParser},
		{"otlp", parserNameOTLP, 55680, NewOTLPReceiverParser},

		// contrib receivers
		{"carbon", parserNameCarbon, 2003, NewCarbonReceiverParser},
		{"collectd", parserNameCollectd, 8081, NewCollectdReceiverParser},
		{"sapm", parserNameSAPM, 7276, NewSAPMReceiverParser},
	} {

		t.Run("Builder", func(t *testing.T) {
			// test
			builder := tt.builder(tt.receiverName, map[interface{}]interface{}{})

			// verify
			assert.Equal(t, tt.parserName, builder.ParserName())
		})

		t.Run("DefaultPort", func(t *testing.T) {
			// prepare
			builder := tt.builder(tt.receiverName, map[interface{}]interface{}{})

			// test
			ports, err := builder.Ports(ctxWithLogger)

			// verify
			assert.NoError(t, err)
			assert.Len(t, ports, 1)
			assert.EqualValues(t, tt.parserDefaultPort, ports[0].Port)
			assert.Equal(t, tt.receiverName, ports[0].Name)
		})

		t.Run("OverridePort", func(t *testing.T) {
			// prepare
			builder := tt.builder(tt.receiverName, map[interface{}]interface{}{
				"endpoint": "0.0.0.0:65535",
			})

			// test
			ports, err := builder.Ports(ctxWithLogger)

			// verify
			assert.NoError(t, err)
			assert.Len(t, ports, 1)
			assert.EqualValues(t, 65535, ports[0].Port)
			assert.Equal(t, tt.receiverName, ports[0].Name)
		})
	}
}
