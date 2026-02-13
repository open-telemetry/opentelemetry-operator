// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticProfile(t *testing.T) {
	profile := NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256})
	assert.Equal(t, uint16(tls.VersionTLS12), profile.MinTLSVersion())
	assert.Equal(t, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_AES_128_GCM_SHA256}, profile.CipherSuites())
	assert.Equal(t, []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_AES_128_GCM_SHA256"}, profile.CipherSuiteNames())
	assert.Equal(t, "1.2", profile.MinTLSVersionOTEL())
}

func TestStaticProfileTLS13ReturnsNilCiphers(t *testing.T) {
	// TLS 1.3 cipher suites are not configurable in Go, so CipherSuites() and CipherSuiteNames() should return nil
	profile := NewStaticTLSProfile(tls.VersionTLS13, []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384})
	assert.Equal(t, uint16(tls.VersionTLS13), profile.MinTLSVersion())
	assert.Nil(t, profile.CipherSuites(), "TLS 1.3 should return nil for CipherSuites")
	assert.Nil(t, profile.CipherSuiteNames(), "TLS 1.3 should return nil for CipherSuiteNames")
	assert.Equal(t, "1.3", profile.MinTLSVersionOTEL())
}
