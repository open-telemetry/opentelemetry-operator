package version

import (
	"testing"

	"gotest.tools/assert"
)

func TestDefaultOpenTelemetryService(t *testing.T) {
	assert.Equal(t, "0.0.0", DefaultOpenTelemetryService())
}

func TestCurrentOpenTelemetryService(t *testing.T) {
	otelSvc = "0.0.2" // set during the build
	defer func() {
		otelSvc = ""
	}()
	assert.Equal(t, "0.0.2", Get().OpenTelemetryService)
}
