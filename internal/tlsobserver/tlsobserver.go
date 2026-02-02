// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tlsobserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

const (
	// clusterAPIServerName is the name of the cluster-wide APIServer config.
	clusterAPIServerName = "cluster"
	// resyncPeriod is how often we re-check the APIServer config.
	resyncPeriod = 5 * time.Minute
)

var log = ctrl.Log.WithName("tls-observer")

// +kubebuilder:rbac:groups=config.openshift.io,resources=apiservers,verbs=get;watch

// TLSObserver watches the OpenShift APIServer config and extracts TLS settings.
type TLSObserver struct {
	client client.Client

	mu            sync.RWMutex
	minTLSVersion uint16
	cipherSuites  []uint16
	cipherNames   []string // Go/IANA format cipher names
	initialized   bool
}

var _ manager.Runnable = (*TLSObserver)(nil)
var _ manager.LeaderElectionRunnable = (*TLSObserver)(nil)
var _ components.TLSProfileProvider = (*TLSObserver)(nil)
var _ components.TLSProfile = (*TLSObserver)(nil)

// NewTLSObserver creates a new TLS observer.
func NewTLSObserver(client client.Client) *TLSObserver {
	return &TLSObserver{
		client: client,
	}
}

// Start implements manager.Runnable.
func (t *TLSObserver) Start(ctx context.Context) error {
	log.Info("Starting TLS security profile observer")

	// Initial fetch
	if err := t.FetchAndUpdateTLSConfig(ctx); err != nil {
		// Log but don't fail - use defaults
		log.Error(err, "Failed to fetch initial TLS config, using defaults")
		t.setDefaults()
	}

	// Use polling approach since controller-runtime client doesn't expose Watch directly
	// and setting up informers for a single resource is complex
	return t.pollLoop(ctx)
}

// pollLoop polls for APIServer config changes at regular intervals.
func (t *TLSObserver) pollLoop(ctx context.Context) error {
	ticker := time.NewTicker(resyncPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("TLS observer shutting down")
			return nil
		case <-ticker.C:
			if err := t.FetchAndUpdateTLSConfig(ctx); err != nil {
				log.Error(err, "Failed to update TLS config during poll")
			}
		}
	}
}

// FetchAndUpdateTLSConfig fetches the APIServer config and updates TLS settings.
// This can be called directly for synchronous initialization before starting the manager.
func (t *TLSObserver) FetchAndUpdateTLSConfig(ctx context.Context) error {
	apiServer := &configv1.APIServer{}
	err := t.client.Get(ctx, client.ObjectKey{Name: clusterAPIServerName}, apiServer)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("APIServer config not found, using defaults")
			t.setDefaults()
			return nil
		}
		return fmt.Errorf("failed to get APIServer config: %w", err)
	}

	return t.updateFromAPIServer(apiServer)
}

// updateFromAPIServer extracts TLS settings from the APIServer config.
func (t *TLSObserver) updateFromAPIServer(apiServer *configv1.APIServer) error {
	profile := apiServer.Spec.TLSSecurityProfile
	if profile == nil {
		log.Info("No TLS security profile configured, using intermediate profile")
		spec := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		return t.applyTLSSpec(spec.Ciphers, string(spec.MinTLSVersion))
	}

	spec, err := getTLSProfileSpec(profile)
	if err != nil {
		return err
	}

	return t.applyTLSSpec(spec.Ciphers, string(spec.MinTLSVersion))
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

// applyTLSSpec converts and applies TLS settings.
func (t *TLSObserver) applyTLSSpec(ciphers []string, minVersion string) error {
	// Convert TLS version name to uint16
	tlsVersion, err := parseTLSVersion(minVersion)
	if err != nil {
		return fmt.Errorf("invalid TLS version %s: %w", minVersion, err)
	}

	// Convert to Go format (e.g., "VersionTLS12")

	var cipherIDs []uint16
	var cipherNames []string

	// TLS 1.3 cipher suites are not configurable in Go - they're always enabled.
	// Only parse cipher suites for TLS 1.2 and earlier.
	if tlsVersion < tls.VersionTLS13 {
		cipherIDs = parseCipherSuites(ciphers)
		// Convert OpenSSL cipher names to Go/IANA format
		cipherNames = convertToGoCipherNames(ciphers)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	oldVersion := t.minTLSVersion
	oldCiphers := t.cipherNames

	t.minTLSVersion = tlsVersion
	t.cipherSuites = cipherIDs
	t.cipherNames = cipherNames
	t.initialized = true

	if oldVersion != tlsVersion || !stringSliceEqual(oldCiphers, cipherNames) {
		log.Info("TLS configuration updated",
			"minVersion", tls.VersionName(tlsVersion),
			"cipherCount", len(cipherIDs))
	}

	return nil
}

// setDefaults sets default TLS configuration (Intermediate profile).
func (t *TLSObserver) setDefaults() {
	spec := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
	if spec == nil {
		// Fallback to TLS 1.2 with default Go ciphers
		t.mu.Lock()
		t.minTLSVersion = tls.VersionTLS12
		t.cipherSuites = nil
		t.initialized = true
		t.mu.Unlock()
		return
	}

	_ = t.applyTLSSpec(spec.Ciphers, string(spec.MinTLSVersion))
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
// TLS observer doesn't need leader election as it's read-only.
func (t *TLSObserver) NeedLeaderElection() bool {
	return false
}

// GetTLSConfig returns a function that can be used to configure TLS.
func (t *TLSObserver) GetTLSConfig() func(*tls.Config) {
	return func(cfg *tls.Config) {
		t.mu.RLock()
		defer t.mu.RUnlock()

		if !t.initialized {
			// Use defaults if not initialized
			cfg.MinVersion = tls.VersionTLS12
			return
		}

		cfg.MinVersion = t.minTLSVersion
		if len(t.cipherSuites) > 0 {
			cfg.CipherSuites = t.cipherSuites
		}
	}
}

// MinTLSVersion returns the current minimum TLS version as a Go crypto/tls constant.
func (t *TLSObserver) MinTLSVersion() uint16 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.initialized {
		return tls.VersionTLS12
	}
	return t.minTLSVersion
}

// GetCipherSuites returns the current cipher suites.
func (t *TLSObserver) GetCipherSuites() []uint16 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cipherSuites
}

// IsInitialized returns true if the observer has been initialized.
func (t *TLSObserver) IsInitialized() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.initialized
}

// GetTLSProfile returns the current TLS profile for injection into collector components.
// Returns nil if the observer has not been initialized.
func (t *TLSObserver) GetTLSProfile() components.TLSProfile {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.initialized {
		return nil
	}
	return t
}

// MinTLSVersionOTEL returns the minimum TLS version in OpenTelemetry collector format (e.g., "1.2").
func (t *TLSObserver) MinTLSVersionOTEL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return components.TLSVersionToCollectorFormat(t.minTLSVersion)
}

// CipherSuites returns the cipher suites as Go crypto/tls constants.
// For TLS 1.3, this returns nil as cipher suites are not configurable.
func (t *TLSObserver) CipherSuites() []uint16 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// TLS 1.3 cipher suites are not configurable in Go
	if t.minTLSVersion >= tls.VersionTLS13 {
		return nil
	}
	return t.cipherSuites
}

// CipherSuiteNames returns the cipher suite names in Go/IANA format.
func (t *TLSObserver) CipherSuiteNames() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// TLS 1.3 cipher suites are not configurable in Go
	if t.minTLSVersion >= tls.VersionTLS13 {
		return nil
	}
	if len(t.cipherNames) == 0 {
		return nil
	}
	return t.cipherNames
}

// Ensure TLSObserver implements TLSProfileProvider.
var _ components.TLSProfileProvider = (*TLSObserver)(nil)

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// parseCipherSuites converts OpenSSL-style cipher names (as used in OpenShift TLS profiles)
// to Go's crypto/tls package constants. Uses openSSLToIANACiphersMap and ciphers as source of truth.
func parseCipherSuites(names []string) []uint16 {
	suites := make([]uint16, 0, len(names))
	for _, name := range names {
		// First convert OpenSSL name to IANA name
		ianaName, ok := openSSLToIANACiphersMap[name]
		if !ok {
			continue
		}
		// Then look up the Go constant
		if suite, ok := ciphers[ianaName]; ok {
			suites = append(suites, suite)
		}
	}
	return suites
}

var ciphers = map[string]uint16{
	"TLS_RSA_WITH_RC4_128_SHA":                      tls.TLS_RSA_WITH_RC4_128_SHA,
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":          tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_AES_128_GCM_SHA256":                        tls.TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384":                        tls.TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256":                  tls.TLS_CHACHA20_POLY1305_SHA256,
}

// openSSLToIANACiphersMap maps OpenSSL cipher suite names to IANA names
// ref: https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
var openSSLToIANACiphersMap = map[string]string{
	// TLS 1.3 ciphers - not configurable in go 1.13, all of them are used in TLSv1.3 flows
	"TLS_AES_128_GCM_SHA256":       "TLS_AES_128_GCM_SHA256",       // 0x13,0x01
	"TLS_AES_256_GCM_SHA384":       "TLS_AES_256_GCM_SHA384",       // 0x13,0x02
	"TLS_CHACHA20_POLY1305_SHA256": "TLS_CHACHA20_POLY1305_SHA256", // 0x13,0x03

	// TLS 1.2
	"ECDHE-ECDSA-AES128-GCM-SHA256": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",       // 0xC0,0x2B
	"ECDHE-RSA-AES128-GCM-SHA256":   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",         // 0xC0,0x2F
	"ECDHE-ECDSA-AES256-GCM-SHA384": "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",       // 0xC0,0x2C
	"ECDHE-RSA-AES256-GCM-SHA384":   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",         // 0xC0,0x30
	"ECDHE-ECDSA-CHACHA20-POLY1305": "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256", // 0xCC,0xA9
	"ECDHE-RSA-CHACHA20-POLY1305":   "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",   // 0xCC,0xA8
	"ECDHE-ECDSA-AES128-SHA256":     "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",       // 0xC0,0x23
	"ECDHE-RSA-AES128-SHA256":       "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",         // 0xC0,0x27
	"AES128-GCM-SHA256":             "TLS_RSA_WITH_AES_128_GCM_SHA256",               // 0x00,0x9C
	"AES256-GCM-SHA384":             "TLS_RSA_WITH_AES_256_GCM_SHA384",               // 0x00,0x9D
	"AES128-SHA256":                 "TLS_RSA_WITH_AES_128_CBC_SHA256",               // 0x00,0x3C

	// TLS 1
	"ECDHE-ECDSA-AES128-SHA": "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA", // 0xC0,0x09
	"ECDHE-RSA-AES128-SHA":   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",   // 0xC0,0x13
	"ECDHE-ECDSA-AES256-SHA": "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", // 0xC0,0x0A
	"ECDHE-RSA-AES256-SHA":   "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",   // 0xC0,0x14

	// SSL 3
	"AES128-SHA":   "TLS_RSA_WITH_AES_128_CBC_SHA",  // 0x00,0x2F
	"AES256-SHA":   "TLS_RSA_WITH_AES_256_CBC_SHA",  // 0x00,0x35
	"DES-CBC3-SHA": "TLS_RSA_WITH_3DES_EDE_CBC_SHA", // 0x00,0x0A
}

// parseTLSVersion converts TLS version strings to Go's crypto/tls constants.
// Accepts both OpenShift format ("VersionTLS12") and standard format ("TLSv1.2").
func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "VersionTLS10", "TLSv1.0":
		return tls.VersionTLS10, nil
	case "VersionTLS11", "TLSv1.1":
		return tls.VersionTLS11, nil
	case "VersionTLS12", "TLSv1.2":
		return tls.VersionTLS12, nil
	case "VersionTLS13", "TLSv1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unknown TLS version: %s", version)
	}
}

// convertToGoCipherNames converts OpenSSL cipher names to Go/IANA format.
// OpenShift uses OpenSSL names like "ECDHE-RSA-AES128-GCM-SHA256",
// but Go's crypto/tls and the collector expect IANA names like "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256".
// Uses openSSLToIANACiphersMap as the source of truth.
func convertToGoCipherNames(opensslNames []string) []string {
	goNames := make([]string, 0, len(opensslNames))
	for _, name := range opensslNames {
		if ianaName, ok := openSSLToIANACiphersMap[name]; ok {
			goNames = append(goNames, ianaName)
		}
	}
	return goNames
}
