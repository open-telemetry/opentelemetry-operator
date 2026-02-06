// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tlsobserver

import (
	"context"
	"crypto/tls"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetTLSProfile(t *testing.T) {
	scheme := runtime.NewScheme()
	err := configv1.AddToScheme(scheme)
	require.NoError(t, err)

	tests := []struct {
		name           string
		apiServer      *configv1.APIServer
		expectError    bool
		expectedMinVer uint16
		expectCiphers  bool // true if we expect ciphers, false for TLS 1.3
	}{
		{
			name:           "no APIServer config returns defaults",
			apiServer:      nil,
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
		{
			name: "intermediate profile",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileIntermediateType,
					},
				},
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
		{
			name: "modern profile with TLS 1.3",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileModernType,
					},
				},
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS13,
			expectCiphers:  false, // TLS 1.3 returns nil for cipher suites
		},
		{
			name: "old profile",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileOldType,
					},
				},
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS10,
			expectCiphers:  true,
		},
		{
			name: "custom profile",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileCustomType,
						Custom: &configv1.CustomTLSProfile{
							TLSProfileSpec: configv1.TLSProfileSpec{
								Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256"},
								MinTLSVersion: configv1.VersionTLS12,
							},
						},
					},
				},
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
		{
			name: "nil TLS profile uses intermediate",
			apiServer: &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec:       configv1.APIServerSpec{},
			},
			expectError:    false,
			expectedMinVer: tls.VersionTLS12,
			expectCiphers:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objs []runtime.Object
			if tt.apiServer != nil {
				objs = append(objs, tt.apiServer)
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				Build()

			observer := NewTLSObserver(fakeClient)
			profile, err := observer.GetTLSProfile(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, profile)
			assert.Equal(t, tt.expectedMinVer, profile.MinTLSVersion())

			if tt.expectCiphers {
				assert.NotEmpty(t, profile.CipherSuites(), "expected cipher suites for TLS version < 1.3")
			} else {
				// TLS 1.3 should return nil for cipher suites
				assert.Nil(t, profile.CipherSuites())
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []uint16
	}{
		{
			name:     "OpenSSL format ciphers",
			input:    []string{"ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES256-GCM-SHA384"},
			expected: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:     "mixed valid and invalid",
			input:    []string{"ECDHE-RSA-AES128-GCM-SHA256", "INVALID-CIPHER"},
			expected: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []uint16{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCipherSuites(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
