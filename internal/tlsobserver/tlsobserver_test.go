// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tlsobserver

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTLSProfile(t *testing.T) {
	tests := []struct {
		name            string
		initialized     bool
		minTLSVersion   uint16   // Go crypto/tls constant
		cipherNames     []string // Go/IANA format
		expectNil       bool
		expectedCiphers []string
	}{
		{
			name:        "uninitialized returns nil",
			initialized: false,
			expectNil:   true,
		},
		{
			name:            "initialized returns settings",
			initialized:     true,
			minTLSVersion:   tls.VersionTLS12,
			cipherNames:     []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
			expectNil:       false,
			expectedCiphers: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
		},
		{
			name:            "initialized with TLS 1.3 returns nil for ciphers",
			initialized:     true,
			minTLSVersion:   tls.VersionTLS13,
			cipherNames:     []string{},
			expectNil:       false,
			expectedCiphers: nil, // TLS 1.3 returns nil for cipher names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &TLSObserver{
				initialized:   tt.initialized,
				minTLSVersion: tt.minTLSVersion,
				cipherNames:   tt.cipherNames,
			}

			profile := observer.GetTLSProfile()

			if tt.expectNil {
				assert.Nil(t, profile)
			} else {
				assert.NotNil(t, profile)
				assert.Equal(t, tt.expectedCiphers, profile.CipherSuiteNames())
			}
		})
	}
}
