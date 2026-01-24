// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func TestIsCollectorVersionUnupgradable(t *testing.T) {
	// Store original map and restore after test
	originalVersions := unupgradableCollectorVersions
	defer func() { unupgradableCollectorVersions = originalVersions }()

	tests := []struct {
		name               string
		version            string
		unupgradableMap    map[string]string
		expectedUnupgrable bool
		expectedMessage    string
	}{
		{
			name:               "normal version returns false",
			version:            "0.100.0",
			unupgradableMap:    map[string]string{},
			expectedUnupgrable: false,
			expectedMessage:    "",
		},
		{
			name:    "unupgradable version returns true with message",
			version: "0.99.0",
			unupgradableMap: map[string]string{
				"0.99.0": "Breaking changes in configuration format require manual migration.",
			},
			expectedUnupgrable: true,
			expectedMessage:    "Breaking changes in configuration format require manual migration.",
		},
		{
			name:    "unupgradable version returns true with empty message",
			version: "0.98.0",
			unupgradableMap: map[string]string{
				"0.98.0": "",
			},
			expectedUnupgrable: true,
			expectedMessage:    "",
		},
		{
			name:    "version not in map returns false",
			version: "0.97.0",
			unupgradableMap: map[string]string{
				"0.99.0": "Some message",
			},
			expectedUnupgrable: false,
			expectedMessage:    "",
		},
		{
			name:               "empty version returns false",
			version:            "",
			unupgradableMap:    map[string]string{},
			expectedUnupgrable: false,
			expectedMessage:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unupgradableCollectorVersions = tt.unupgradableMap

			isUnupgradable, message := IsCollectorVersionUnupgradable(tt.version)

			assert.Equal(t, tt.expectedUnupgrable, isUnupgradable)
			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}

func TestExtractVersionFromImage(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "full image with tag",
			image:    "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:v1.0.0",
			expected: "v1.0.0",
		},
		{
			name:     "image with port and tag",
			image:    "localhost:5000/myimage:v2.0.0",
			expected: "v2.0.0",
		},
		{
			name:     "just version string",
			image:    "v1.0.0",
			expected: "v1.0.0",
		},
		{
			name:     "image without tag",
			image:    "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java",
			expected: "latest",
		},
		{
			name:     "empty string",
			image:    "",
			expected: "",
		},
		{
			name:     "semver without v prefix",
			image:    "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "image with sha digest",
			image:    "myregistry.io/image@sha256:abc123",
			expected: "abc123", // Extracts the portion after the last ":"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromImage(tt.image)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInstrumentationVersionUnupgradable(t *testing.T) {
	// Store original map and restore after test
	originalVersions := unupgradableInstrumentationVersions
	defer func() { unupgradableInstrumentationVersions = originalVersions }()

	tests := []struct {
		name               string
		language           constants.InstrumentationLanguage
		imageOrVersion     string
		unupgradableMap    map[constants.InstrumentationLanguage]map[string]string
		expectedUnupgrable bool
		expectedMessage    string
	}{
		{
			name:               "normal version returns false",
			language:           constants.InstrumentationLanguageJava,
			imageOrVersion:     "ghcr.io/org/java:v1.0.0",
			unupgradableMap:    map[constants.InstrumentationLanguage]map[string]string{},
			expectedUnupgrable: false,
			expectedMessage:    "",
		},
		{
			name:           "unupgradable version returns true with message",
			language:       constants.InstrumentationLanguageJava,
			imageOrVersion: "ghcr.io/org/java:v1.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageJava: {
					"v1.0.0": "Breaking changes in Java agent.",
				},
			},
			expectedUnupgrable: true,
			expectedMessage:    "Breaking changes in Java agent.",
		},
		{
			name:           "unupgradable version with just version string",
			language:       constants.InstrumentationLanguagePython,
			imageOrVersion: "v2.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguagePython: {
					"v2.0.0": "Python SDK breaking change.",
				},
			},
			expectedUnupgrable: true,
			expectedMessage:    "Python SDK breaking change.",
		},
		{
			name:           "different language is not affected",
			language:       constants.InstrumentationLanguageNodeJS,
			imageOrVersion: "ghcr.io/org/nodejs:v1.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageJava: {
					"v1.0.0": "Java only.",
				},
			},
			expectedUnupgrable: false,
			expectedMessage:    "",
		},
		{
			name:           "empty message returns true with empty message",
			language:       constants.InstrumentationLanguageGo,
			imageOrVersion: "ghcr.io/org/go:v3.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageGo: {
					"v3.0.0": "",
				},
			},
			expectedUnupgrable: true,
			expectedMessage:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unupgradableInstrumentationVersions = tt.unupgradableMap

			isUnupgradable, message := IsInstrumentationVersionUnupgradable(tt.language, tt.imageOrVersion)

			assert.Equal(t, tt.expectedUnupgrable, isUnupgradable)
			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}
