// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package tlsobserver provides utilities for converting OpenShift TLS security profiles
// to formats usable by the OpenTelemetry Operator and its operands (collectors).
//
// # Architecture
//
// The operator uses github.com/openshift/controller-runtime-common/pkg/tls for TLS profile
// management. This package provides:
//   - FetchAPIServerTLSProfile: fetches the TLS profile from the cluster's APIServer CR
//   - NewTLSConfigFromProfile: converts profile to Go's crypto/tls configuration
//   - SecurityProfileWatcher: watches for TLS profile changes
//
// When the TLS profile changes, the SecurityProfileWatcher triggers a graceful restart
// of the operator (via context cancellation). This means operands don't need to watch
// for profile changes - they simply use the profile that was current when the operator
// started, and will get the new profile after the operator restarts.
//
// This package (tlsobserver) provides TLSProfileFromSpec() to convert the profile spec
// into the format needed for operand configuration (OpenTelemetry Collector YAML format).
package tlsobserver

import (
	"crypto/tls"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/crypto"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// +kubebuilder:rbac:groups=config.openshift.io,resources=apiservers,verbs=get;list;watch

// TLSProfileFromSpec converts a configv1.TLSProfileSpec to a components.TLSProfile.
// This is used to convert the TLS profile fetched by controller-runtime-common's
// FetchAPIServerTLSProfile into the format needed for operand configuration.
//
// The returned TLSProfile provides methods like MinTLSVersionOTEL() and CipherSuiteNames()
// that return values in OpenTelemetry Collector configuration format.
//
// Note: This function does not watch for profile changes. The operator restarts when
// the TLS profile changes (via SecurityProfileWatcher), so operands always receive
// the current profile without needing dynamic updates.
func TLSProfileFromSpec(spec configv1.TLSProfileSpec) (components.TLSProfile, error) {
	return buildTLSProfile(spec.Ciphers, string(spec.MinTLSVersion))
}

// buildTLSProfile creates a TLSProfile from the given ciphers and TLS version.
func buildTLSProfile(opensslCiphers []string, minVersion string) (components.TLSProfile, error) {
	// Use library-go's TLSVersion to convert version string to uint16
	tlsVersion, err := crypto.TLSVersion(minVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid TLS version %s: %w", minVersion, err)
	}

	var cipherIDs []uint16
	// TLS 1.3 cipher suites are not configurable in Go - they're always enabled.
	// Only parse cipher suites for TLS 1.2 and earlier.
	if tlsVersion < tls.VersionTLS13 {
		cipherIDs = parseCipherSuites(opensslCiphers)
	}

	return components.NewStaticTLSProfile(tlsVersion, cipherIDs), nil
}

// parseCipherSuites converts OpenSSL-style cipher names (as used in OpenShift TLS profiles)
// to Go's crypto/tls package constants using library-go's crypto functions.
func parseCipherSuites(opensslCiphers []string) []uint16 {
	// Convert OpenSSL cipher names to IANA format using library-go
	ianaCiphers := crypto.OpenSSLToIANACipherSuites(opensslCiphers)

	// Convert IANA names to Go uint16 constants
	suites := make([]uint16, 0, len(ianaCiphers))
	for _, name := range ianaCiphers {
		suite, err := crypto.CipherSuite(name)
		if err != nil {
			// Skip unknown ciphers (some may not be supported by Go's crypto/tls)
			continue
		}
		suites = append(suites, suite)
	}
	return suites
}
