// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tlsobserver

import (
	"crypto/tls"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLSProfileFromSpec(t *testing.T) {
	tests := []struct {
		name           string
		spec           configv1.TLSProfileSpec
		expectError    bool
		expectedMinVer uint16
		expectCiphers  bool // true if we expect ciphers, false for TLS 1.3
	}{
		{
			name:           "intermediate profile spec",
			spec:           *configv1.TLSProfiles[configv1.TLSProfileIntermediateType],
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
		{
			name:           "modern profile spec with TLS 1.3",
			spec:           *configv1.TLSProfiles[configv1.TLSProfileModernType],
			expectError:    false,
			expectedMinVer: tls.VersionTLS13,
			expectCiphers:  false, // TLS 1.3 returns nil for cipher suites
		},
		{
			name:           "old profile spec",
			spec:           *configv1.TLSProfiles[configv1.TLSProfileOldType],
			expectError:    false,
			expectedMinVer: tls.VersionTLS10,
			expectCiphers:  true,
		},
		{
			name: "custom profile spec",
			spec: configv1.TLSProfileSpec{
				Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256"},
				MinTLSVersion: configv1.VersionTLS12,
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := TLSProfileFromSpec(tt.spec)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, profile)
			assert.Equal(t, tt.expectedMinVer, profile.MinTLSVersion())

			if tt.expectCiphers {
				assert.NotEmpty(t, profile.CipherSuites(), "expected cipher suites for TLS version < 1.3")
			} else {
				// TLS 1.3 should return nil for cipher suites
				assert.Nil(t, profile.CipherSuites())
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []uint16
	}{
		{
			name:     "OpenSSL format ciphers",
			input:    []string{"ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES256-GCM-SHA384"},
			expected: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:     "mixed valid and invalid",
			input:    []string{"ECDHE-RSA-AES128-GCM-SHA256", "INVALID-CIPHER"},
			expected: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []uint16{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCipherSuites(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
