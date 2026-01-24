// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
