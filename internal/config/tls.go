// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"fmt"

	k8sapiflag "k8s.io/component-base/cli/flag"
)

type TLSConfig struct {
	MinVersion   string
	CipherSuites []string
}

// ApplyTLSConfig get the option from command argument (tlsConfig), check the validity through k8s apiflag
// and set the config for webhook server.
// refer to https://pkg.go.dev/k8s.io/component-base/cli/flag
func (tlsOpt TLSConfig) ApplyTLSConfig(cfg *tls.Config) error {
	// TLSVersion helper function returns the TLS Version ID for the version name passed.
	tlsVersion, err := k8sapiflag.TLSVersion(tlsOpt.MinVersion)
	if err != nil {
		return fmt.Errorf("TLS version invalid: %w", err)
	}

	// TLSCipherSuites helper function returns a list of cipher suite IDs from the cipher suite names passed.
	cipherSuiteIDs, err := k8sapiflag.TLSCipherSuites(tlsOpt.CipherSuites)
	if err != nil {
		return fmt.Errorf("failed to convert TLS cipher suite name to ID: %w", err)
	}
	cfg.MinVersion = tlsVersion
	cfg.CipherSuites = cipherSuiteIDs
	return nil
}
