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

// TLSConfig holds OTel-format TLS settings that can be injected into component configs.
type TLSConfig struct {
	Ciphers    []string `mapstructure:"cipher_suites,omitempty"`
	MinVersion string   `mapstructure:"min_version,omitempty"`
	Curves     []string `mapstructure:"curve_preferences,omitempty"`
}

// ApplyTLSProfileDefaults sets MinVersion, Ciphers, and Curves from the given
// TLS profile when the corresponding fields are not already configured.
func (t *TLSConfig) ApplyTLSProfileDefaults(profile TLSProfile) {
	if t == nil || profile == nil {
		return
	}
	if t.MinVersion == "" && profile.MinTLSVersionOTEL() != "" {
		t.MinVersion = profile.MinTLSVersionOTEL()
	}
	if t.Ciphers == nil && len(profile.CipherSuiteNames()) > 0 {
		t.Ciphers = profile.CipherSuiteNames()
	}
	if t.Curves == nil {
		if names, _ := profile.CurveNamesOTEL(); len(names) > 0 {
			t.Curves = names
		}
	}
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
	// CurveIDs returns the supported groups as Go crypto/tls constants.
	// Unlike cipher suites, supported groups remain configurable under TLS 1.3.
	CurveIDs() []tls.CurveID
	// CurveNamesOTEL returns the supported group names in OpenTelemetry
	// collector format (e.g., "P256"), and the Go spellings of groups with no
	// collector translation; those groups are dropped from the first return.
	// Unlike cipher suites, supported groups remain configurable under TLS 1.3.
	CurveNamesOTEL() ([]string, []string)
}

// TLSVersionToCollectorFormat converts a TLS version constant to collector format string.
func TLSVersionToCollectorFormat(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return "1.2"
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
	curves     []tls.CurveID
}

func NewStaticTLSProfile(minVersion uint16, ciphers []uint16, curves []tls.CurveID) StaticTLSProfile {
	return StaticTLSProfile{
		minVersion: minVersion,
		ciphers:    ciphers,
		curves:     curves,
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

func (p StaticTLSProfile) CurveIDs() []tls.CurveID {
	// Unlike cipher suites, supported groups remain configurable under TLS 1.3.
	return p.curves
}

// curveNamesOTEL maps Go crypto/tls CurveID values to the OpenTelemetry
// collector's curve_preferences spellings. The collector's vocabulary cannot
// be derived from Go: CurveID.String() yields "CurveP256", which matches
// neither the OpenShift nor the collector spelling.
var curveNamesOTEL = map[tls.CurveID]string{
	tls.X25519:         "X25519",
	tls.CurveP256:      "P256",
	tls.CurveP384:      "P384",
	tls.CurveP521:      "P521",
	tls.X25519MLKEM768: "X25519MLKEM768",
}

// CurveNamesOTEL returns the supported group names in OpenTelemetry collector
// format (e.g., "P256"), and the Go spellings of groups that have no
// collector curve_preferences translation in curveNamesOTEL; such groups are
// dropped from the first return. A newer Go release can add a group the
// collector does not know, so callers log the second return. Unlike cipher
// suites, supported groups remain configurable under TLS 1.3.
func (p StaticTLSProfile) CurveNamesOTEL() (names, unmapped []string) {
	for _, c := range p.curves {
		if name, ok := curveNamesOTEL[c]; ok {
			names = append(names, name)
		} else {
			unmapped = append(unmapped, c.String())
		}
	}
	return names, unmapped
}
