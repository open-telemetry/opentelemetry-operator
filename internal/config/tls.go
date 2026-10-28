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
	UseClusterProfile bool
	// Configures TLS in operands created by the operator.
	ConfigureOperands bool
	MinVersion        string
	CipherSuites      []string
	CurvePreferences  []int32
}

// ApplyTLSConfig get the option from command argument (tlsConfig), check the validity through k8s apiflag
// and set the config for webhook server.
// refer to https://pkg.go.dev/k8s.io/component-base/cli/flag
func (tlsOpt TLSConfig) ApplyTLSConfig(cfg *tls.Config) error {
	tlsVersion, err := k8sapiflag.TLSVersion(tlsOpt.MinVersion)
	if err != nil {
		return fmt.Errorf("TLS version invalid: %w", err)
	}

	cipherSuiteIDs, err := k8sapiflag.TLSCipherSuites(tlsOpt.CipherSuites)
	if err != nil {
		return fmt.Errorf("failed to convert TLS cipher suite name to ID: %w", err)
	}

	curveIDs, err := k8sapiflag.TLSCurvePreferences(tlsOpt.CurvePreferences)
	if err != nil {
		return fmt.Errorf("failed to convert TLS curve preference to ID: %w", err)
	}

	cfg.MinVersion = tlsVersion
	cfg.CipherSuites = cipherSuiteIDs
	cfg.CurvePreferences = curveIDs
	return nil
}
