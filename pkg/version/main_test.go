package version

import (
	"testing"

	"gotest.tools/assert"
)

func TestDefaultOpenTelemetryCollector(t *testing.T) {
	assert.Equal(t, "0.0.0", DefaultOpenTelemetryCollector())
}

func TestCurrentOpenTelemetryCollector(t *testing.T) {
	otelCol = "0.0.2" // set during the build
	defer func() {
		otelCol = ""
	}()
	assert.Equal(t, "0.0.2", Get().OpenTelemetryCollector)
}
