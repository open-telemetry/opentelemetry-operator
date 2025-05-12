// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestComponentType(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		name     string
		expected string
	}{
		{"regular case", "myreceiver", "myreceiver"},
		{"named instance", "myreceiver/custom", "myreceiver"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test and verify
			assert.Equal(t, tt.expected, components.ComponentType(tt.name))
		})
	}
}

func TestReceiverParsePortFromEndpoint(t *testing.T) {
	for _, tt := range []struct {
		desc          string
		endpoint      string
		expected      int
		errorExpected bool
	}{
		{"regular case", "http://localhost:1234", 1234, false},
		{"absolute with path", "http://localhost:1234/server-status?auto", 1234, false},
		{"no protocol", "0.0.0.0:1234", 1234, false},
		{"just port", ":1234", 1234, false},
		{"no port at all", "http://localhost", 0, true},
		{"overflow", "0.0.0.0:2147483648", 0, true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			val, err := components.PortFromEndpoint(tt.endpoint)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.EqualValues(t, tt.expected, val, "wrong port from endpoint %s: %d", tt.endpoint, val)
		})
	}
}
