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

func TestTACertificateVolumesCertManager(t *testing.T) {
	// With cert-manager (useCertManager defaulted), a single operator-managed Secret is mounted at
	// /tls without any subPath mapping.
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	serverVolumes, serverMounts := TAServerCertificateVolumes(ta)
	require.Len(t, serverVolumes, 1)
	assert.Equal(t, naming.TAServerCertificate("test"), serverVolumes[0].Name)
	require.NotNil(t, serverVolumes[0].Secret)
	assert.Equal(t, naming.TAServerCertificateSecretName("test"), serverVolumes[0].Secret.SecretName)
	assert.Nil(t, serverVolumes[0].Secret.Items)
	require.Len(t, serverMounts, 1)
	assert.Equal(t, constants.TACollectorTLSDirPath, serverMounts[0].MountPath)
	assert.Empty(t, serverMounts[0].SubPath)

	clientVolumes, clientMounts := TAClientCertificateVolumes(ta, "test")
	require.Len(t, clientVolumes, 1)
	assert.Equal(t, naming.TAClientCertificateSecretName("test"), clientVolumes[0].Secret.SecretName)
	require.Len(t, clientMounts, 1)
	assert.Equal(t, constants.TACollectorTLSDirPath, clientMounts[0].MountPath)
}

func TestTACertificateVolumesUserProvidedDefaultKeys(t *testing.T) {
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

	serverVolumes, serverMounts := TAServerCertificateVolumes(ta)
	// CA is bundled in the leaf Secret, so a single volume backs all three files.
	require.Len(t, serverVolumes, 1)
	assert.Equal(t, "my-server-secret", serverVolumes[0].Secret.SecretName)
	assert.ElementsMatch(t, toMountSpecs(serverVolumes, serverMounts), []mountSpec{
		{secretName: "my-server-secret", subPath: constants.TACollectorCAFileName, path: tlsPath(constants.TACollectorCAFileName)},
		{secretName: "my-server-secret", subPath: constants.TACollectorTLSCertFileName, path: tlsPath(constants.TACollectorTLSCertFileName)},
		{secretName: "my-server-secret", subPath: constants.TACollectorTLSKeyFileName, path: tlsPath(constants.TACollectorTLSKeyFileName)},
	})

	clientVolumes, clientMounts := TAClientCertificateVolumes(ta, "test")
	require.Len(t, clientVolumes, 1)
	assert.Equal(t, "my-client-secret", clientVolumes[0].Secret.SecretName)
	assert.ElementsMatch(t, toMountSpecs(clientVolumes, clientMounts), []mountSpec{
		{secretName: "my-client-secret", subPath: constants.TACollectorCAFileName, path: tlsPath(constants.TACollectorCAFileName)},
		{secretName: "my-client-secret", subPath: constants.TACollectorTLSCertFileName, path: tlsPath(constants.TACollectorTLSCertFileName)},
		{secretName: "my-client-secret", subPath: constants.TACollectorTLSKeyFileName, path: tlsPath(constants.TACollectorTLSKeyFileName)},
	})
}

func TestTACertificateVolumesUserProvidedCustomKeys(t *testing.T) {
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

	serverVolumes, serverMounts := TAServerCertificateVolumes(ta)
	// user's arbitrary keys are projected onto the fixed /tls filenames via subPath
	assert.ElementsMatch(t, toMountSpecs(serverVolumes, serverMounts), []mountSpec{
		{secretName: "my-server-secret", subPath: constants.TACollectorCAFileName, path: tlsPath(constants.TACollectorCAFileName)},
		{secretName: "my-server-secret", subPath: "server.pem", path: tlsPath(constants.TACollectorTLSCertFileName)},
		{secretName: "my-server-secret", subPath: "server-key.pem", path: tlsPath(constants.TACollectorTLSKeyFileName)},
	})
}

func TestTACertificateVolumesSeparateCA(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: new(false),
		TLS: &v1beta1.TargetAllocatorTLS{
			CertificateAuthorityCertificate: &v1beta1.CertificateReference{SecretName: "ca-secret", DataKeyCertificate: "ca.pem"},
			ServerCertificate:               &v1beta1.CertificateReference{SecretName: "server-secret"},
			ClientCertificate:               &v1beta1.CertificateReference{SecretName: "client-secret"},
		},
	}

	serverVolumes, serverMounts := TAServerCertificateVolumes(ta)
	// CA comes from a distinct Secret, so two volumes are created.
	require.Len(t, serverVolumes, 2)
	assert.ElementsMatch(t, toMountSpecs(serverVolumes, serverMounts), []mountSpec{
		{secretName: "ca-secret", subPath: "ca.pem", path: tlsPath(constants.TACollectorCAFileName)},
		{secretName: "server-secret", subPath: constants.TACollectorTLSCertFileName, path: tlsPath(constants.TACollectorTLSCertFileName)},
		{secretName: "server-secret", subPath: constants.TACollectorTLSKeyFileName, path: tlsPath(constants.TACollectorTLSKeyFileName)},
	})
}

func TestTACertificateVolumesSameSecretForAllRefs(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Name = "test"
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: new(false),
		TLS: &v1beta1.TargetAllocatorTLS{
			CertificateAuthorityCertificate: &v1beta1.CertificateReference{SecretName: "shared", DataKeyCertificate: "ca.crt"},
			ServerCertificate:               &v1beta1.CertificateReference{SecretName: "shared", DataKeyCertificate: "tls.crt", DataKeyKey: "tls.key"},
			ClientCertificate:               &v1beta1.CertificateReference{SecretName: "shared"},
		},
	}

	serverVolumes, serverMounts := TAServerCertificateVolumes(ta)
	// One shared Secret backs the CA, certificate and key, so only a single volume is created.
	require.Len(t, serverVolumes, 1)
	assert.Equal(t, "shared", serverVolumes[0].Secret.SecretName)
	assert.ElementsMatch(t, toMountSpecs(serverVolumes, serverMounts), []mountSpec{
		{secretName: "shared", subPath: "ca.crt", path: tlsPath(constants.TACollectorCAFileName)},
		{secretName: "shared", subPath: "tls.crt", path: tlsPath(constants.TACollectorTLSCertFileName)},
		{secretName: "shared", subPath: "tls.key", path: tlsPath(constants.TACollectorTLSKeyFileName)},
	})
}

// mountSpec is a flattened view of a VolumeMount that also resolves the backing Secret name, used to
// assert on mounts independently of the generated volume names.
type mountSpec struct {
	secretName string
	subPath    string
	path       string
}

// toMountSpecs resolves each VolumeMount back to the Secret that backs its volume, so tests can assert
// on (secret, key, path) tuples without depending on generated volume names.
func toMountSpecs(volumes []corev1.Volume, mounts []corev1.VolumeMount) []mountSpec {
	byName := map[string]string{}
	for _, v := range volumes {
		if v.Secret != nil {
			byName[v.Name] = v.Secret.SecretName
		}
	}
	specs := make([]mountSpec, 0, len(mounts))
	for _, m := range mounts {
		specs = append(specs, mountSpec{secretName: byName[m.Name], subPath: m.SubPath, path: m.MountPath})
	}
	return specs
}

func tlsPath(file string) string {
	return constants.TACollectorTLSDirPath + "/" + file
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
			name: "user-provided, separate CA is allowed",
			ta: &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{
				Enabled:        true,
				UseCertManager: new(false),
				TLS: &v1beta1.TargetAllocatorTLS{
					ServerCertificate:               serverRef,
					ClientCertificate:               clientRef,
					CertificateAuthorityCertificate: &v1beta1.CertificateReference{SecretName: "ca"},
				},
			}}},
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
