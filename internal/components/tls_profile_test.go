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
		expectedMinVersion  uint16
		expectedGolangVer   string
		expectedOTELVer     string
		expectedCiphers     []uint16
		expectedCipherNames []string
	}{
		{
			name:                "TLS 1.2 with custom ciphers",
			minVersion:          tls.VersionTLS12,
			ciphers:             []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256},
			expectedMinVersion:  tls.VersionTLS12,
			expectedGolangVer:   "TLS 1.2",
			expectedOTELVer:     "1.2",
			expectedCiphers:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256},
			expectedCipherNames: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_AES_128_GCM_SHA256"},
		},
		{
			name:                "TLS 1.3 suppresses ciphers (crypto/tls invariant)",
			minVersion:          tls.VersionTLS13,
			ciphers:             []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384},
			expectedMinVersion:  tls.VersionTLS13,
			expectedGolangVer:   "TLS 1.3",
			expectedOTELVer:     "1.3",
			expectedCiphers:     nil,
			expectedCipherNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := NewStaticTLSProfile(tt.minVersion, tt.ciphers)
			assert.Equal(t, tt.expectedMinVersion, profile.MinTLSVersion())
			assert.Equal(t, tt.expectedGolangVer, profile.MinTLSVersionGolang())
			assert.Equal(t, tt.expectedOTELVer, profile.MinTLSVersionOTEL())
			assert.Equal(t, tt.expectedCiphers, profile.CipherSuites())
			assert.Equal(t, tt.expectedCipherNames, profile.CipherSuiteNames())
		})
	}
}

func TestStaticTLSProfileProvider(t *testing.T) {
	profile := NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256})
	provider := StaticTLSProfileProvider{Profile: profile}

	res, err := provider.GetTLSProfile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, profile, res)
}

func TestApplyTLSProfileDefaults(t *testing.T) {
	defaultProfile := NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256})
	tls13Profile := NewStaticTLSProfile(tls.VersionTLS13, []uint16{tls.TLS_AES_128_GCM_SHA256})

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
			},
		},
		{
			name:          "Does not populate Ciphers if profile has no ciphers (TLS 1.3)",
			initialConfig: &TLSConfig{},
			profile:       tls13Profile,
			expectedConfig: &TLSConfig{
				MinVersion: "1.3",
				Ciphers:    nil,
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
