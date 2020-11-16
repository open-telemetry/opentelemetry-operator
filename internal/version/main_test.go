package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", OpenTelemetryCollector())
}

func TestVersionFromBuild(t *testing.T) {
	// prepare
	otelCol = "0.0.2" // set during the build
	defer func() {
		otelCol = ""
	}()

	assert.Equal(t, otelCol, OpenTelemetryCollector())
	assert.Contains(t, Get().String(), otelCol)
}
