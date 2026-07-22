// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func IsTAMTLSEnabled(ta *v1alpha1.TargetAllocator) bool {
	return ta != nil && ta.Spec.Mtls != nil && ta.Spec.Mtls.Enabled
}

func IsTAMTLSCertManagerEnabled(ta *v1alpha1.TargetAllocator, cfg config.Config) bool {
	if !IsTAMTLSEnabled(ta) {
		return false
	}
	if ta.Spec.Mtls.UseCertManager != nil && !*ta.Spec.Mtls.UseCertManager {
		return false
	}
	return cfg.CertManagerAvailability == certmanager.Available
}

// IsTAMTLSUserProvided reports whether mTLS is enabled with user-provided certificates
// (i.e. cert-manager is explicitly disabled). In this mode the operator mounts the Secrets
// referenced in the TLS block instead of provisioning cert-manager Certificates.
func IsTAMTLSUserProvided(ta *v1alpha1.TargetAllocator) bool {
	return IsTAMTLSEnabled(ta) &&
		ta.Spec.Mtls.UseCertManager != nil &&
		!*ta.Spec.Mtls.UseCertManager
}

// TAServerCertificateVolume builds the volume that mounts the target allocator's server certificate.
// When certificates are user-provided it references the user's Secret and maps its keys to the fixed
// filenames the target allocator config expects; otherwise it references the cert-manager Secret.
func TAServerCertificateVolume(ta *v1alpha1.TargetAllocator) corev1.Volume {
	return taCertificateVolume(
		ta,
		naming.TAServerCertificate(ta.Name),
		naming.TAServerCertificateSecretName(ta.Name),
		taServerCertificateReference(ta),
	)
}

// TAClientCertificateVolume builds the volume that mounts the collector's client certificate.
// When certificates are user-provided it references the user's Secret and maps its keys to the fixed
// filenames the prometheus receiver config expects; otherwise it references the cert-manager Secret.
func TAClientCertificateVolume(ta *v1alpha1.TargetAllocator, otelcolName string) corev1.Volume {
	return taCertificateVolume(
		ta,
		naming.TAClientCertificate(otelcolName),
		naming.TAClientCertificateSecretName(otelcolName),
		taClientCertificateReference(ta),
	)
}

// taCertificateVolume builds a Secret-backed volume for an mTLS certificate. When certificates are
// user-provided, the volume references the user's Secret with Items mapping the user's data keys
// onto the fixed /tls filenames (the CA certificate is expected to be bundled in the same Secret,
// keyed by ca.crt). Otherwise it references the cert-manager-managed Secret, which already stores
// the standard keys.
func taCertificateVolume(ta *v1alpha1.TargetAllocator, volumeName, certManagerSecretName string, ref *v1beta1.CertificateReference) corev1.Volume {
	if !IsTAMTLSUserProvided(ta) || ref == nil {
		return corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certManagerSecretName,
				},
			},
		}
	}

	items := []corev1.KeyToPath{
		{Key: dataKeyCertificate(ref), Path: constants.TACollectorTLSCertFileName},
		{Key: dataKeyKey(ref), Path: constants.TACollectorTLSKeyFileName},
		{Key: constants.TACollectorCAFileName, Path: constants.TACollectorCAFileName},
	}

	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ref.SecretName,
				Items:      items,
			},
		},
	}
}

// taTLS returns the user-provided TLS configuration, or nil when it isn't set.
func taTLS(ta *v1alpha1.TargetAllocator) *v1beta1.TargetAllocatorTLS {
	if ta == nil || ta.Spec.Mtls == nil {
		return nil
	}
	return ta.Spec.Mtls.TLS
}

func taServerCertificateReference(ta *v1alpha1.TargetAllocator) *v1beta1.CertificateReference {
	if tls := taTLS(ta); tls != nil {
		return tls.ServerCertificate
	}
	return nil
}

func taClientCertificateReference(ta *v1alpha1.TargetAllocator) *v1beta1.CertificateReference {
	if tls := taTLS(ta); tls != nil {
		return tls.ClientCertificate
	}
	return nil
}

func caCertificateReference(ta *v1alpha1.TargetAllocator) *v1beta1.CertificateReference {
	if tls := taTLS(ta); tls != nil {
		return tls.CertificateAuthorityCertificate
	}
	return nil
}

// dataKeyCertificate returns the Secret data key holding the certificate, defaulting to tls.crt.
func dataKeyCertificate(ref *v1beta1.CertificateReference) string {
	if ref != nil && ref.DataKeyCertificate != "" {
		return ref.DataKeyCertificate
	}
	return constants.TACollectorTLSCertFileName
}

// dataKeyKey returns the Secret data key holding the private key, defaulting to tls.key.
func dataKeyKey(ref *v1beta1.CertificateReference) string {
	if ref != nil && ref.DataKeyKey != "" {
		return ref.DataKeyKey
	}
	return constants.TACollectorTLSKeyFileName
}

// ValidateTAMTLS validates the target allocator mTLS configuration. When mTLS relies on cert-manager
// it requires cert-manager to be available. When cert-manager is disabled it requires the user to
// provide the server and client certificate Secrets, and rejects the currently-unsupported separate
// CA certificate reference (the CA is expected to be bundled in the leaf Secrets).
func ValidateTAMTLS(ta *v1alpha1.TargetAllocator, certManagerAvailable bool) error {
	if !IsTAMTLSEnabled(ta) {
		return nil
	}

	if !IsTAMTLSUserProvided(ta) {
		if !certManagerAvailable {
			return errors.New("mTLS is enabled with useCertManager but cert-manager is not available; install cert-manager and restart the operator, or set useCertManager to false")
		}
		return nil
	}

	// User-provided certificates: both leaf certificates must be referenced.
	if taServerCertificateReference(ta) == nil || taClientCertificateReference(ta) == nil {
		return errors.New("mTLS is enabled with useCertManager set to false; tls.serverCertificate and tls.clientCertificate must both reference a Secret")
	}
	if caCertificateReference(ta) != nil {
		return errors.New("mTLS with a separate tls.certificateAuthorityCertificate is not supported yet; bundle the CA certificate under the ca.crt key inside the server and client Secrets instead")
	}
	return nil
}
