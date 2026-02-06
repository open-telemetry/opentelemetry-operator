// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"context"
	"crypto/tls"
)

// TLSProfileProvider provides TLS settings for component configuration.
// This interface is implemented by TLSObserver to provide cluster-wide TLS settings.
type TLSProfileProvider interface {
	// GetTLSProfile fetches the TLS profile from the cluster.
	// This is a blocking call that fetches the current TLS security profile.
	// Returns nil profile if TLS profile is not configured or not available.
	GetTLSProfile(ctx context.Context) (TLSProfile, error)
}

// TLSProfile holds the TLS configuration to inject into collector components.
// These settings are derived from the cluster's TLS security profile.
type TLSProfile interface {
	// MinTLSVersionOTEL returns the minimum TLS version in OpenTelemetry collector format (e.g., "1.2").
	MinTLSVersionOTEL() string
	// MinTLSVersion returns the minimum TLS version as a Go crypto/tls constant.
	MinTLSVersion() uint16
	// CipherSuites returns the cipher suites as Go crypto/tls constants.
	// For TLS 1.3, this returns nil as cipher suites are not configurable.
	CipherSuites() []uint16
	// CipherSuiteNames returns the cipher suite names in Go/IANA format.
	// For TLS 1.3, this returns nil as cipher suites are not configurable.
	CipherSuiteNames() []string
}

// TLSVersionToCollectorFormat converts a TLS version constant to collector format string.
func TLSVersionToCollectorFormat(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return "1.2" // Default to TLS 1.2
	}
}

var _ TLSProfileProvider = (*StaticTLSProfileProvider)(nil)

type StaticTLSProfileProvider struct {
	Profile TLSProfile
}

func (d StaticTLSProfileProvider) GetTLSProfile(_ context.Context) (TLSProfile, error) {
	return d.Profile, nil
}

var _ TLSProfile = (*StaticTLSProfile)(nil)

type StaticTLSProfile struct {
	minVersion uint16
	ciphers    []uint16
}

func NewStaticTLSProfile(minVersion uint16, ciphers []uint16) StaticTLSProfile {
	return StaticTLSProfile{
		minVersion: minVersion,
		ciphers:    ciphers,
	}
}

func (p StaticTLSProfile) MinTLSVersionGolang() string {
	return tls.VersionName(p.minVersion)
}

func (p StaticTLSProfile) MinTLSVersionOTEL() string {
	return TLSVersionToCollectorFormat(p.minVersion)
}

func (p StaticTLSProfile) MinTLSVersion() uint16 {
	return p.minVersion
}

func (p StaticTLSProfile) CipherSuites() []uint16 {
	// TLS 1.3 cipher suites are not configurable in Go
	if p.minVersion >= tls.VersionTLS13 {
		return nil
	}
	return p.ciphers
}

func (p StaticTLSProfile) CipherSuiteNames() []string {
	// TLS 1.3 cipher suites are not configurable in Go
	if p.minVersion >= tls.VersionTLS13 {
		return nil
	}
	names := make([]string, 0, len(p.ciphers))
	for _, c := range p.ciphers {
		names = append(names, tls.CipherSuiteName(c))
	}
	return names
}
