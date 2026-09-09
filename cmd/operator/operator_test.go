// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestFallbackToEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      string
		wantPort  int32
		wantIPs   []string
		wantError bool
	}{
		{
			name:      "valid env vars",
			host:      "10.96.0.1",
			port:      "443",
			wantPort:  443,
			wantIPs:   []string{"10.96.0.1"},
			wantError: false,
		},
		{
			name:      "missing host",
			host:      "",
			port:      "443",
			wantError: true,
		},
		{
			name:      "missing port",
			host:      "10.96.0.1",
			port:      "",
			wantError: true,
		},
		{
			name:      "invalid port",
			host:      "10.96.0.1",
			port:      "not-a-number",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			if tt.host != "" {
				os.Setenv("KUBERNETES_SERVICE_HOST", tt.host)
				defer os.Unsetenv("KUBERNETES_SERVICE_HOST")
			} else {
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
			}
			if tt.port != "" {
				os.Setenv("KUBERNETES_SERVICE_PORT", tt.port)
				defer os.Unsetenv("KUBERNETES_SERVICE_PORT")
			} else {
				os.Unsetenv("KUBERNETES_SERVICE_PORT")
			}

			cfg := &config.Config{}
			err := fallbackToEnvVars(cfg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPort, cfg.Internal.KubeAPIServerPort)
				assert.Equal(t, tt.wantIPs, cfg.Internal.KubeAPIServerIPs)
			}
		})
	}
}
