// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

func TestTargetAllocatorFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", TargetAllocator())
}

func TestTargetAllocatorVersionFromBuild(t *testing.T) {
	// prepare
	targetAllocator = "0.0.2" // set during the build
	defer func() {
		targetAllocator = ""
	}()

	assert.Equal(t, targetAllocator, TargetAllocator())
	assert.Contains(t, Get().String(), targetAllocator)
}

func TestAutoInstrumentationJavaFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", AutoInstrumentationJava())
}

func TestAutoInstrumentationNodeJSFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", AutoInstrumentationNodeJS())
}

func TestAutoInstrumentationPythonFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", AutoInstrumentationPython())
}
