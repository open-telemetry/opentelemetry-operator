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
		{"ipv6 loopback", "[::1]:4317", 4317, false},
		{"ipv6 global address", "[2001:db8::1]:4317", 4317, false},
		{"ipv6 private address", "[fd00:10:244::5]:4318", 4318, false},
		{"ipv6 regression - port inside address", "[fd00:10:244::5]:4317", 4317, false},
		{"just port", ":1234", 1234, false},
		{"no port at all", "http://localhost", 0, true},
		{"overflow", "0.0.0.0:2147483648", 0, true},
		{"ipv6 wildcard", "[::]:4317", 4317, false},
		{"ipv6 URL with path", "http://[::1]:4317/metrics", 4317, false},
		{"ipv6 without port", "[::1]", 0, true},
		{"empty port", "localhost:", 0, true},
		{"zero port", "localhost:0", 0, true},
		{"negative port", "0.0.0.0:-1", 0, true},
		{"invalid port", "localhost:abc", 0, true},
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

func TestPortFromEndpointMissingPort(t *testing.T) {
	_, err := components.PortFromEndpoint("invalid-endpoint-without-port")
	assert.ErrorIs(t, err, components.PortNotFoundErr)
}
