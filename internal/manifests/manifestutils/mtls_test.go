// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func TestIsTAMTLSEnabledTrue(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	assert.True(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSEnabledFalse(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: false}

	assert.False(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSEnabledNilTA(t *testing.T) {
	assert.False(t, IsTAMTLSEnabled(nil))
}

func TestIsTAMTLSEnabledNilMtls(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}

	assert.False(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSCertManagerEnabled(t *testing.T) {
	boolTrue := true
	boolFalse := false

	tests := []struct {
		name     string
		ta       *v1alpha1.TargetAllocator
		cfg      config.Config
		expected bool
	}{
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager defaulting to true",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: true,
		},
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager explicitly true",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: &boolTrue}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: true,
		},
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager false",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: &boolFalse}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
		{
			name:     "mTLS enabled, cert-manager not available",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.NotAvailable},
			expected: false,
		},
		{
			name:     "mTLS disabled",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: false}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
		{
			name:     "nil TA",
			ta:       nil,
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTAMTLSCertManagerEnabled(tt.ta, tt.cfg))
		})
	}
}

func TestIsTAMTLSUserProvided(t *testing.T) {
	tests := []struct {
		name     string
		ta       *v1alpha1.TargetAllocator
		expected bool
	}{
		{
			name:     "mTLS enabled, useCertManager defaulted",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			expected: false,
		},
		{
			name:     "mTLS enabled, useCertManager true",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: new(true)}}},
			expected: false,
		},
		{
			name:     "mTLS enabled, useCertManager false",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: new(false)}}},
			expected: true,
		},
		{
			name:     "mTLS disabled, useCertManager false",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: false, UseCertManager: new(false)}}},
			expected: false,
		},
		{
			name:     "nil TA",
			ta:       nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTAMTLSUserProvided(tt.ta))
		})
	}
}

func TestTACertificateVolumeCertManager(t *testing.T) {
	// With cert-manager (useCertManager defaulted), volumes reference the operator-managed Secrets
	// without any Items mapping.
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	server := TAServerCertificateVolume(ta)
	assert.Equal(t, naming.TAServerCertificate("test"), server.Name)
	require.NotNil(t, server.Secret)
	assert.Equal(t, naming.TAServerCertificateSecretName("test"), server.Secret.SecretName)
	assert.Nil(t, server.Secret.Items)

	client := TAClientCertificateVolume(ta, "test")
	assert.Equal(t, naming.TAClientCertificate("test"), client.Name)
	require.NotNil(t, client.Secret)
	assert.Equal(t, naming.TAClientCertificateSecretName("test"), client.Secret.SecretName)
	assert.Nil(t, client.Secret.Items)
}

func TestTACertificateVolumeUserProvidedDefaultKeys(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: new(false),
		TLS: &v1beta1.TargetAllocatorTLS{
			ServerCertificate: &v1beta1.CertificateReference{SecretName: "my-server-secret"},
			ClientCertificate: &v1beta1.CertificateReference{SecretName: "my-client-secret"},
		},
	}

	server := TAServerCertificateVolume(ta)
	// volume name is stable regardless of the certificate source
	assert.Equal(t, naming.TAServerCertificate("test"), server.Name)
	require.NotNil(t, server.Secret)
	assert.Equal(t, "my-server-secret", server.Secret.SecretName)
	assert.ElementsMatch(t, []corev1.KeyToPath{
		{Key: constants.TACollectorTLSCertFileName, Path: constants.TACollectorTLSCertFileName},
		{Key: constants.TACollectorTLSKeyFileName, Path: constants.TACollectorTLSKeyFileName},
		{Key: constants.TACollectorCAFileName, Path: constants.TACollectorCAFileName},
	}, server.Secret.Items)

	client := TAClientCertificateVolume(ta, "test")
	require.NotNil(t, client.Secret)
	assert.Equal(t, "my-client-secret", client.Secret.SecretName)
	assert.ElementsMatch(t, []corev1.KeyToPath{
		{Key: constants.TACollectorTLSCertFileName, Path: constants.TACollectorTLSCertFileName},
		{Key: constants.TACollectorTLSKeyFileName, Path: constants.TACollectorTLSKeyFileName},
		{Key: constants.TACollectorCAFileName, Path: constants.TACollectorCAFileName},
	}, client.Secret.Items)
}

func TestTACertificateVolumeUserProvidedCustomKeys(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: new(false),
		TLS: &v1beta1.TargetAllocatorTLS{
			ServerCertificate: &v1beta1.CertificateReference{
				SecretName:         "my-server-secret",
				DataKeyCertificate: "server.pem",
				DataKeyKey:         "server-key.pem",
			},
			ClientCertificate: &v1beta1.CertificateReference{SecretName: "my-client-secret"},
		},
	}

	server := TAServerCertificateVolume(ta)
	require.NotNil(t, server.Secret)
	// user's arbitrary keys are mapped onto the fixed /tls filenames
	assert.ElementsMatch(t, []corev1.KeyToPath{
		{Key: "server.pem", Path: constants.TACollectorTLSCertFileName},
		{Key: "server-key.pem", Path: constants.TACollectorTLSKeyFileName},
		{Key: constants.TACollectorCAFileName, Path: constants.TACollectorCAFileName},
	}, server.Secret.Items)
}

func TestValidateTAMTLS(t *testing.T) {
	serverRef := &v1beta1.CertificateReference{SecretName: "server"}
	clientRef := &v1beta1.CertificateReference{SecretName: "client"}

	tests := []struct {
		name                 string
		ta                   *v1alpha1.TargetAllocator
		certManagerAvailable bool
		expectedErr          string
	}{
		{
			name: "mTLS disabled",
			ta:   &v1alpha1.TargetAllocator{},
		},
		{
			name:                 "cert-manager path, available",
			ta:                   &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			certManagerAvailable: true,
		},
		{
			name:        "cert-manager path, not available",
			ta:          &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			expectedErr: "cert-manager is not available",
		},
		{
			name: "user-provided, both refs set",
			ta: &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{
				Enabled:        true,
				UseCertManager: new(false),
				TLS:            &v1beta1.TargetAllocatorTLS{ServerCertificate: serverRef, ClientCertificate: clientRef},
			}}},
		},
		{
			name: "user-provided, missing client ref",
			ta: &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{
				Enabled:        true,
				UseCertManager: new(false),
				TLS:            &v1beta1.TargetAllocatorTLS{ServerCertificate: serverRef},
			}}},
			expectedErr: "tls.serverCertificate and tls.clientCertificate must both reference a Secret",
		},
		{
			name: "user-provided, no TLS block",
			ta: &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{
				Enabled:        true,
				UseCertManager: new(false),
			}}},
			expectedErr: "tls.serverCertificate and tls.clientCertificate must both reference a Secret",
		},
		{
			name: "user-provided, separate CA not supported",
			ta: &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{
				Enabled:        true,
				UseCertManager: new(false),
				TLS: &v1beta1.TargetAllocatorTLS{
					ServerCertificate:               serverRef,
					ClientCertificate:               clientRef,
					CertificateAuthorityCertificate: &v1beta1.CertificateReference{SecretName: "ca"},
				},
			}}},
			expectedErr: "not supported yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTAMTLS(tt.ta, tt.certManagerAvailable)
			if tt.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
