// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLSVersionToCollectorFormat(t *testing.T) {
	tests := []struct {
		name     string
		version  uint16
		expected string
	}{
		{
			name:     "TLS 1.0",
			version:  tls.VersionTLS10,
			expected: "1.0",
		},
		{
			name:     "TLS 1.1",
			version:  tls.VersionTLS11,
			expected: "1.1",
		},
		{
			name:     "TLS 1.2",
			version:  tls.VersionTLS12,
			expected: "1.2",
		},
		{
			name:     "TLS 1.3",
			version:  tls.VersionTLS13,
			expected: "1.3",
		},
		{
			name:     "Default fallback for unmapped version",
			version:  0x0300, // SSL 3.0 or any non-standard constant
			expected: "1.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TLSVersionToCollectorFormat(tt.version))
		})
	}
}

func TestStaticTLSProfile(t *testing.T) {
	tests := []struct {
		name                string
		minVersion          uint16
		ciphers             []uint16
		curves              []tls.CurveID
		expectedMinVersion  uint16
		expectedGolangVer   string
		expectedOTELVer     string
		expectedCiphers     []uint16
		expectedCipherNames []string
		expectedCurveIDs    []tls.CurveID
		expectedCurveNames  []string
	}{
		{
			name:                "TLS 1.2 with custom ciphers",
			minVersion:          tls.VersionTLS12,
			ciphers:             []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256},
			curves:              []tls.CurveID{tls.X25519, tls.CurveP256},
			expectedMinVersion:  tls.VersionTLS12,
			expectedGolangVer:   "TLS 1.2",
			expectedOTELVer:     "1.2",
			expectedCiphers:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256},
			expectedCipherNames: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_AES_128_GCM_SHA256"},
			expectedCurveIDs:    []tls.CurveID{tls.X25519, tls.CurveP256},
			expectedCurveNames:  []string{"X25519", "P256"},
		},
		{
			name:                "TLS 1.3 suppresses ciphers (crypto/tls invariant)",
			minVersion:          tls.VersionTLS13,
			ciphers:             []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384},
			curves:              []tls.CurveID{tls.X25519, tls.CurveP384},
			expectedMinVersion:  tls.VersionTLS13,
			expectedGolangVer:   "TLS 1.3",
			expectedOTELVer:     "1.3",
			expectedCiphers:     nil,
			expectedCipherNames: nil,
			// Unlike cipher suites, supported groups remain configurable under TLS 1.3.
			expectedCurveIDs:   []tls.CurveID{tls.X25519, tls.CurveP384},
			expectedCurveNames: []string{"X25519", "P384"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := NewStaticTLSProfile(tt.minVersion, tt.ciphers, tt.curves)
			assert.Equal(t, tt.expectedMinVersion, profile.MinTLSVersion())
			assert.Equal(t, tt.expectedGolangVer, profile.MinTLSVersionGolang())
			assert.Equal(t, tt.expectedOTELVer, profile.MinTLSVersionOTEL())
			assert.Equal(t, tt.expectedCiphers, profile.CipherSuites())
			assert.Equal(t, tt.expectedCipherNames, profile.CipherSuiteNames())
			assert.Equal(t, tt.expectedCurveIDs, profile.CurveIDs())
			curveNames, unmappedCurveNames := profile.CurveNamesOTEL()
			assert.Equal(t, tt.expectedCurveNames, curveNames)
			assert.Empty(t, unmappedCurveNames)
		})
	}
}

func TestCurveNamesOTEL(t *testing.T) {
	// A group Go knows but the collector does not, as a future Go release
	// could introduce.
	futureGroup := tls.CurveID(0x9999)

	tests := []struct {
		name             string
		curves           []tls.CurveID
		expectedNames    []string
		expectedUnmapped []string
	}{
		{
			name:   "No curves",
			curves: nil,
		},
		{
			name:          "All curves mapped",
			curves:        []tls.CurveID{tls.X25519, tls.CurveP256},
			expectedNames: []string{"X25519", "P256"},
		},
		{
			name:             "Unmapped curve is dropped and reported",
			curves:           []tls.CurveID{tls.X25519, futureGroup},
			expectedNames:    []string{"X25519"},
			expectedUnmapped: []string{futureGroup.String()},
		},
		{
			// All groups from library-go's TLS profiles must translate; the collector
			// spelling cannot be derived from CurveID.String(). SecP256r1MLKEM768 and
			// SecP384r1MLKEM1024 are Go-supported groups the collector does not accept
			// in curve_preferences yet; they are intentionally absent from curveNamesOTEL
			// and get dropped and reported as unmapped.
			name: "Go groups the collector does not know",
			curves: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
				tls.X25519MLKEM768,
				tls.SecP256r1MLKEM768,
				tls.SecP384r1MLKEM1024,
			},
			expectedNames: []string{
				"X25519",
				"P256",
				"P384",
				"P521",
				"X25519MLKEM768",
			},
			expectedUnmapped: []string{
				tls.SecP256r1MLKEM768.String(),
				tls.SecP384r1MLKEM1024.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := NewStaticTLSProfile(tls.VersionTLS12, nil, tt.curves)
			names, unmapped := profile.CurveNamesOTEL()
			assert.Equal(t, tt.expectedNames, names)
			assert.Equal(t, tt.expectedUnmapped, unmapped)
		})
	}
}

func TestStaticTLSProfileProvider(t *testing.T) {
	profile := NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}, nil)
	provider := StaticTLSProfileProvider{Profile: profile}

	res, err := provider.GetTLSProfile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, profile, res)
}

func TestApplyTLSProfileDefaults(t *testing.T) {
	defaultProfile := NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}, []tls.CurveID{tls.X25519, tls.CurveP256})
	tls13Profile := NewStaticTLSProfile(tls.VersionTLS13, []uint16{tls.TLS_AES_128_GCM_SHA256}, []tls.CurveID{tls.X25519, tls.CurveP384})

	tests := []struct {
		name           string
		initialConfig  *TLSConfig
		profile        TLSProfile
		expectedConfig *TLSConfig
	}{
		{
			name:           "Nil receiver does not panic",
			initialConfig:  nil,
			profile:        defaultProfile,
			expectedConfig: nil,
		},
		{
			name:           "Nil profile is a no-op",
			initialConfig:  &TLSConfig{MinVersion: "1.1"},
			profile:        nil,
			expectedConfig: &TLSConfig{MinVersion: "1.1"},
		},
		{
			name:          "Applies defaults when empty",
			initialConfig: &TLSConfig{},
			profile:       defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.2",
				Ciphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
				Curves:     []string{"X25519", "P256"},
			},
		},
		{
			name: "Preserves user-defined MinVersion and Ciphers",
			initialConfig: &TLSConfig{
				MinVersion: "1.3",
				Ciphers:    []string{"TLS_CUSTOM_CIPHER"},
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.3",
				Ciphers:    []string{"TLS_CUSTOM_CIPHER"},
				Curves:     []string{"X25519", "P256"},
			},
		},
		{
			name: "Applies Ciphers default when only MinVersion is configured",
			initialConfig: &TLSConfig{
				MinVersion: "1.3",
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.3",
				Ciphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
				Curves:     []string{"X25519", "P256"},
			},
		},
		{
			name: "Applies MinVersion default when only Ciphers are configured",
			initialConfig: &TLSConfig{
				Ciphers: []string{"TLS_CUSTOM_CIPHER"},
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.2",
				Ciphers:    []string{"TLS_CUSTOM_CIPHER"},
				Curves:     []string{"X25519", "P256"},
			},
		},
		{
			name: "Preserves explicitly empty Ciphers",
			initialConfig: &TLSConfig{
				Ciphers: []string{},
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.2",
				Ciphers:    []string{},
				Curves:     []string{"X25519", "P256"},
			},
		},
		{
			name:          "Does not populate Ciphers if profile has no ciphers (TLS 1.3)",
			initialConfig: &TLSConfig{},
			profile:       tls13Profile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.3",
				Ciphers:    nil,
				// Curves are still populated: supported groups remain
				// configurable under TLS 1.3, unlike cipher suites.
				Curves: []string{"X25519", "P384"},
			},
		},
		{
			name: "Preserves user-defined Curves",
			initialConfig: &TLSConfig{
				Curves: []string{"X25519"},
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.2",
				Ciphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
				Curves:     []string{"X25519"},
			},
		},
		{
			name: "Preserves explicitly empty Curves",
			initialConfig: &TLSConfig{
				Curves: []string{},
			},
			profile: defaultProfile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.2",
				Ciphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
				Curves:     []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initialConfig
			cfg.ApplyTLSProfileDefaults(tt.profile)
			assert.Equal(t, tt.expectedConfig, cfg)
		})
	}
}
