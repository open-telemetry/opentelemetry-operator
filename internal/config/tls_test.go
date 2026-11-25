// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTLSConfig(t *testing.T) {
	tests := []struct {
		name      string
		tlsConfig TLSConfig
		errWanted string
	}{
		{
			name: "default",
			tlsConfig: TLSConfig{
				MinVersion: "VersionTLS12",
			},
			errWanted: "",
		},
		{
			name: "badTLSVersion",
			tlsConfig: TLSConfig{
				MinVersion: "foo",
			},
			errWanted: `TLS version invalid: unknown tls version "foo"`,
		},
		{
			name: "badCipherSuites",
			tlsConfig: TLSConfig{
				MinVersion:   "VersionTLS12",
				CipherSuites: []string{"foo"},
			},
			errWanted: "failed to convert TLS cipher suite name to ID: Cipher suite foo not supported or doesn't exist",
		},
		{
			name: "validCurves",
			tlsConfig: TLSConfig{
				MinVersion:       "VersionTLS12",
				CurvePreferences: []int32{int32(tls.X25519), int32(tls.CurveP256)},
			},
			errWanted: "",
		},
		{
			name: "badCurve",
			tlsConfig: TLSConfig{
				MinVersion:       "VersionTLS12",
				CurvePreferences: []int32{0},
			},
			errWanted: "failed to convert TLS curve preference to ID: curve preference 0 is out of range (must be 1-65535)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &tls.Config{MinVersion: tls.VersionTLS12}
			err := test.tlsConfig.ApplyTLSConfig(cfg)
			if test.errWanted == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.errWanted)
			}
		})
	}
}

func TestApplyTLSConfigCurvePreferences(t *testing.T) {
	tlsConfig := TLSConfig{
		MinVersion:       "VersionTLS12",
		CurvePreferences: []int32{int32(tls.X25519), int32(tls.CurveP256), int32(tls.CurveP384)},
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	require.NoError(t, tlsConfig.ApplyTLSConfig(cfg))
	assert.Equal(t, []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384}, cfg.CurvePreferences)
}

func TestApplyTLSConfigClusterProfileOverridesManual(t *testing.T) {
	manualConfig := TLSConfig{
		MinVersion:       "VersionTLS12",
		CurvePreferences: []int32{int32(tls.CurveP256)},
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	require.NoError(t, manualConfig.ApplyTLSConfig(cfg))
	assert.Equal(t, []tls.CurveID{tls.CurveP256}, cfg.CurvePreferences)

	// Simulate cluster profile overriding by applying a second config on top,
	// mirroring what buildTLSOptions does in setup.go.
	cfg.CurvePreferences = []tls.CurveID{tls.X25519, tls.CurveP384}
	assert.Equal(t, []tls.CurveID{tls.X25519, tls.CurveP384}, cfg.CurvePreferences)
}

func TestApplyTLSConfigNilCurves(t *testing.T) {
	tlsConfig := TLSConfig{
		MinVersion: "VersionTLS12",
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	require.NoError(t, tlsConfig.ApplyTLSConfig(cfg))
	assert.Nil(t, cfg.CurvePreferences)
}
