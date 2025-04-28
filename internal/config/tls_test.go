// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyTLSConfig(t *testing.T) {
	tests := []struct {
		name      string
		tlsConfig TLSConfig
		errWanted string
	}{
		{
			name: "default",
			tlsConfig: TLSConfig{
				MinVersion: "VersionTLS12",
			},
			errWanted: "",
		},
		{
			name: "badTLSVersion",
			tlsConfig: TLSConfig{
				MinVersion: "foo",
			},
			errWanted: `TLS version invalid: unknown tls version "foo"`,
		},
		{
			name: "badCipherSuites",
			tlsConfig: TLSConfig{
				MinVersion:   "VersionTLS12",
				CipherSuites: []string{"foo"},
			},
			errWanted: "failed to convert TLS cipher suite name to ID: Cipher suite foo not supported or doesn't exist",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.tlsConfig.ApplyTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})
			if test.errWanted == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.errWanted)
			}
		})
	}
}
