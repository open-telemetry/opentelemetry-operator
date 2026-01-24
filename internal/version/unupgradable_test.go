// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func TestSplitImage(t *testing.T) {
	tests := []struct {
		image    string
		wantRepo string
		wantTag  string
	}{
		{"ghcr.io/org/image:v1.0.0", "ghcr.io/org/image", "v1.0.0"},
		{"localhost:5000/myimage:v2.0.0", "localhost:5000/myimage", "v2.0.0"},
		{"ghcr.io/org/image", "ghcr.io/org/image", "latest"},
		{"myimage:1.2.3", "myimage", "1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			repo, tag := splitImage(tt.image)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestIsInstrumentationVersionUnupgradable(t *testing.T) {
	// Store original map and restore after test
	originalVersions := unupgradableInstrumentationVersions
	defer func() { unupgradableInstrumentationVersions = originalVersions }()

	tests := []struct {
		name            string
		language        constants.InstrumentationLanguage
		image           string
		defaultImage    string
		unupgradableMap map[constants.InstrumentationLanguage]map[string]string
		wantBlocked     bool
		wantMessage     string
	}{
		{
			name:            "normal version returns false",
			language:        constants.InstrumentationLanguageJava,
			image:           "ghcr.io/org/java:v1.0.0",
			defaultImage:    "ghcr.io/org/java:v2.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{},
			wantBlocked:     false,
		},
		{
			name:         "unupgradable version returns true with message",
			language:     constants.InstrumentationLanguageJava,
			image:        "ghcr.io/org/java:v1.0.0",
			defaultImage: "ghcr.io/org/java:v2.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageJava: {
					"v1.0.0": "Breaking changes in Java agent.",
				},
			},
			wantBlocked: true,
			wantMessage: "Breaking changes in Java agent.",
		},
		{
			name:         "different repo skips check",
			language:     constants.InstrumentationLanguageJava,
			image:        "my-registry.io/custom/java:v1.0.0",
			defaultImage: "ghcr.io/org/java:v2.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageJava: {
					"v1.0.0": "Should not match.",
				},
			},
			wantBlocked: false,
		},
		{
			name:         "different language is not affected",
			language:     constants.InstrumentationLanguageNodeJS,
			image:        "ghcr.io/org/nodejs:v1.0.0",
			defaultImage: "ghcr.io/org/nodejs:v2.0.0",
			unupgradableMap: map[constants.InstrumentationLanguage]map[string]string{
				constants.InstrumentationLanguageJava: {
					"v1.0.0": "Java only.",
				},
			},
			wantBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unupgradableInstrumentationVersions = tt.unupgradableMap

			isBlocked, message := IsInstrumentationVersionUnupgradable(tt.language, tt.image, tt.defaultImage)

			assert.Equal(t, tt.wantBlocked, isBlocked)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}
