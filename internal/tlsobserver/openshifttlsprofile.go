// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tlsobserver

import (
	"context"
	"crypto/tls"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/crypto"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

const (
	// clusterAPIServerName is the name of the cluster-wide APIServer config.
	clusterAPIServerName = "cluster"
)

var log = ctrl.Log.WithName("tls-observer")

// +kubebuilder:rbac:groups=config.openshift.io,resources=apiservers,verbs=get;watch

// TLSObserver fetches TLS settings from the OpenShift APIServer config.
type TLSObserver struct {
	client client.Client
}

var _ components.TLSProfileProvider = (*TLSObserver)(nil)

// NewTLSObserver creates a new TLS observer.
func NewTLSObserver(client client.Client) *TLSObserver {
	return &TLSObserver{
		client: client,
	}
}

// GetTLSProfile fetches the TLS profile from the cluster's APIServer config.
// This is a blocking call that fetches the current TLS security profile.
// Returns nil profile if TLS profile is not configured or not available.
func (t *TLSObserver) GetTLSProfile(ctx context.Context) (components.TLSProfile, error) {
	apiServer := &configv1.APIServer{}
	err := t.client.Get(ctx, client.ObjectKey{Name: clusterAPIServerName}, apiServer)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("APIServer config not found, using defaults")
			return defaultTLSProfile(), nil
		}
		return nil, fmt.Errorf("failed to get APIServer config: %w", err)
	}

	return t.profileFromAPIServer(apiServer)
}

// profileFromAPIServer extracts TLS settings from the APIServer config and returns a TLSProfile.
func (t *TLSObserver) profileFromAPIServer(apiServer *configv1.APIServer) (components.TLSProfile, error) {
	profile := apiServer.Spec.TLSSecurityProfile
	if profile == nil {
		log.V(1).Info("No TLS security profile configured, using intermediate profile")
		spec := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		return buildTLSProfile(spec.Ciphers, string(spec.MinTLSVersion))
	}

	spec, err := getTLSProfileSpec(profile)
	if err != nil {
		return nil, err
	}

	return buildTLSProfile(spec.Ciphers, string(spec.MinTLSVersion))
}

// getTLSProfileSpec returns the TLS profile spec based on the profile type.
func getTLSProfileSpec(profile *configv1.TLSSecurityProfile) (*configv1.TLSProfileSpec, error) {
	switch profile.Type {
	case configv1.TLSProfileOldType,
		configv1.TLSProfileIntermediateType,
		configv1.TLSProfileModernType:
		return configv1.TLSProfiles[profile.Type], nil
	case configv1.TLSProfileCustomType:
		if profile.Custom == nil {
			return nil, fmt.Errorf("custom TLS profile specified but Custom field is nil")
		}
		return &profile.Custom.TLSProfileSpec, nil
	default:
		return configv1.TLSProfiles[configv1.TLSProfileIntermediateType], nil
	}
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

// defaultTLSProfile returns the default TLS profile (Intermediate).
func defaultTLSProfile() components.TLSProfile {
	spec := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
	if spec == nil {
		// Fallback to TLS 1.2 with default Go ciphers
		return components.NewStaticTLSProfile(tls.VersionTLS12, nil)
	}

	profile, err := buildTLSProfile(spec.Ciphers, string(spec.MinTLSVersion))
	if err != nil {
		return components.NewStaticTLSProfile(tls.VersionTLS12, nil)
	}
	return profile
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
