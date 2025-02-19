// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processors_test

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
)

var logger = logf.Log.WithName("unit-tests")

func TestParserForReturns(t *testing.T) {
	const testComponentName = "test"
	parser := processors.ProcessorFor(testComponentName)
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
	processors.Register(testComponentName, components.NewSinglePortParserBuilder(testComponentName, 9000).MustBuild())
	assert.True(t, processors.IsRegistered(testComponentName))
	parser := processors.ProcessorFor(testComponentName)
	assert.Equal(t, "test", parser.ParserType())
	assert.Equal(t, "__test", parser.ParserName())
	ports, err := parser.Ports(logr.Discard(), testComponentName, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.Equal(t, ports[0].Port, int32(9000))
}

func TestDownstreamParsers(t *testing.T) {
	for _, tt := range []struct {
		desc          string
		processorName string
		parserName    string
	}{
		{"k8sattributes", "k8sattributes", "__k8sattributes"},
		{"resourcedetection", "resourcedetection", "__resourcedetection"},
	} {
		t.Run(tt.processorName, func(t *testing.T) {
			t.Run("builds successfully", func(t *testing.T) {
				// test
				parser := processors.ProcessorFor(tt.processorName)

				// verify
				assert.Equal(t, tt.parserName, parser.ParserName())
			})
			t.Run("bad config errors", func(t *testing.T) {
				// prepare
				parser := processors.ProcessorFor(tt.processorName)

				// test throwing in pure junk
				_, err := parser.Ports(logger, tt.processorName, func() {})

				// verify
				assert.Nil(t, err)
			})

		})
	}
}
