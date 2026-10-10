// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"fmt"

	k8sapiflag "k8s.io/component-base/cli/flag"
)

type TLSConfig struct {
	// Instructs the operator to get the TLS profile from the cluster.
	UseClusterProfile bool `flag:"tls-cluster-profile" env:"TLS_CLUSTER_PROFILE" usage:"Retrieves the TLS profile (min version and ciphers) from the cluster. Supported only on OpenShift clusters. The TLS profile is obtained from APIServer CR." group:"TLS"`
	// Configures TLS in operands created by the operator.
	ConfigureOperands bool     `flag:"tls-configure-operands" env:"TLS_CONFIGURE_OPERANDS" usage:"Configures TLS version and cyphers in operands." group:"TLS"`
	MinVersion        string   `flag:"tls-min-version" env:"TLS_MIN_VERSION" usage:"Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants." group:"TLS"`
	CipherSuites      []string `flag:"tls-cipher-suites" env:"TLS_CIPHER_SUITES" usage:"Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used" group:"TLS"`
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
