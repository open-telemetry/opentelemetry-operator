// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestParserForReturns(t *testing.T) {
	const testComponentName = "test"
	parser := ParserFor(testComponentName)
	assert.Equal(t, "test", parser.ParserType())
	assert.Equal(t, "__test", parser.ParserName())
	ports, err := parser.Ports(logr.Discard(), testComponentName, map[string]interface{}{
		"endpoint": "localhost:9000",
	})
	assert.NoError(t, err)
	assert.Len(t, ports, 0) // Should use the nop parser
}

func TestCanRegister(t *testing.T) {
	const testComponentName = "test"
	registry[testComponentName] = components.NewSinglePortParserBuilder(testComponentName, 9000).MustBuild()
	parser := ParserFor(testComponentName)
	assert.Equal(t, "test", parser.ParserType())
	assert.Equal(t, "__test", parser.ParserName())
	ports, err := parser.Ports(logr.Discard(), testComponentName, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.Equal(t, ports[0].Port, int32(9000))
}

func TestExtensionsComponentParsers(t *testing.T) {
	for _, tt := range []struct {
		exporterName string
		parserName   string
		defaultPort  int32
	}{
		{"health_check", "__health_check", 13133},
	} {
		t.Run(tt.exporterName, func(t *testing.T) {
			t.Run("is registered", func(t *testing.T) {
				_, ok := registry[tt.exporterName]
				assert.True(t, ok)
			})
			t.Run("bad config errors", func(t *testing.T) {
				// prepare
				parser := ParserFor(tt.exporterName)

				// test throwing in pure junk
				_, err := parser.Ports(logr.Discard(), tt.exporterName, func() {})

				// verify
				assert.ErrorContains(t, err, "expected a map, got ")
			})

			t.Run("assigns the expected port", func(t *testing.T) {
				// prepare
				parser := ParserFor(tt.exporterName)

				// test
				ports, err := parser.Ports(logr.Discard(), tt.exporterName, map[string]interface{}{})

				if tt.defaultPort == 0 {
					assert.Len(t, ports, 0)
					return
				}
				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, tt.defaultPort, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.exporterName, tt.defaultPort), ports[0].Name)
			})

			t.Run("allows port to be overridden", func(t *testing.T) {
				// prepare
				parser := ParserFor(tt.exporterName)

				// test
				ports, err := parser.Ports(logr.Discard(), tt.exporterName, map[string]interface{}{
					"endpoint": "0.0.0.0:65535",
				})

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, 65535, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.exporterName, tt.defaultPort), ports[0].Name)
			})
		})
	}
}
