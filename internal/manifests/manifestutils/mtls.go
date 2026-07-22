// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"errors"
	"path/filepath"

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

// TAServerCertificateVolumes builds the volumes and volume mounts that provide the target allocator's
// server certificate (and the CA used to verify collector clients). With cert-manager it mounts the
// operator-managed Secret at /tls; with user-provided certificates it projects each referenced Secret
// key onto the corresponding file under /tls via subPath mounts.
func TAServerCertificateVolumes(ta *v1alpha1.TargetAllocator) ([]corev1.Volume, []corev1.VolumeMount) {
	return taCertificateVolumes(
		ta,
		naming.TAServerCertificate(ta.Name),
		naming.TAServerCertificateSecretName(ta.Name),
		taServerCertificateReference(ta),
	)
}

// TAClientCertificateVolumes builds the volumes and volume mounts that provide the collector's client
// certificate (and the CA used to verify the target allocator server). With cert-manager it mounts the
// operator-managed Secret at /tls; with user-provided certificates it projects each referenced Secret
// key onto the corresponding file under /tls via subPath mounts.
func TAClientCertificateVolumes(ta *v1alpha1.TargetAllocator, otelcolName string) ([]corev1.Volume, []corev1.VolumeMount) {
	return taCertificateVolumes(
		ta,
		naming.TAClientCertificate(otelcolName),
		naming.TAClientCertificateSecretName(otelcolName),
		taClientCertificateReference(ta),
	)
}

// taCertificateVolumes returns the volumes and mounts for one side of the mTLS connection.
//
// In the cert-manager case it mounts the single operator-managed Secret at /tls, which already stores
// the standard ca.crt/tls.crt/tls.key keys.
//
// In the user-provided case (useCertManager=false) it builds one Secret volume per distinct referenced
// Secret name and one subPath VolumeMount per file, so that the leaf certificate/key come from certRef
// and the CA certificate comes from either a dedicated CertificateAuthorityCertificate reference or,
// when that isn't set, from the ca.crt key of the leaf Secret. Because references are independent, the
// same Secret may back the CA, certificate and key.
func taCertificateVolumes(ta *v1alpha1.TargetAllocator, volumeName, certManagerSecretName string, certRef *v1beta1.CertificateReference) ([]corev1.Volume, []corev1.VolumeMount) {
	if !IsTAMTLSUserProvided(ta) || certRef == nil {
		volumes := []corev1.Volume{{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: certManagerSecretName,
				},
			},
		}}
		mounts := []corev1.VolumeMount{{
			Name:      volumeName,
			MountPath: constants.TACollectorTLSDirPath,
		}}
		return volumes, mounts
	}

	// The CA certificate is served from its own reference when provided, otherwise it is expected to
	// be bundled in the leaf Secret under the ca.crt key.
	caRef := caCertificateReference(ta)
	caSecretName := certRef.SecretName
	caKey := constants.TACollectorCAFileName
	if caRef != nil {
		caSecretName = caRef.SecretName
		caKey = dataKeyCertificate(caRef)
	}

	files := []mtlsFile{
		{secretName: caSecretName, key: caKey, path: constants.TACollectorCAFileName},
		{secretName: certRef.SecretName, key: dataKeyCertificate(certRef), path: constants.TACollectorTLSCertFileName},
		{secretName: certRef.SecretName, key: dataKeyKey(certRef), path: constants.TACollectorTLSKeyFileName},
	}

	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	seenSecret := map[string]string{} // secret name -> volume name
	for _, f := range files {
		volName, ok := seenSecret[f.secretName]
		if !ok {
			// Derive a stable, unique volume name per distinct Secret.
			volName = naming.DNSName(naming.Truncate("%s-%d", 63, volumeName, len(volumes)))
			seenSecret[f.secretName] = volName
			volumes = append(volumes, corev1.Volume{
				Name: volName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: f.secretName,
					},
				},
			})
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: filepath.Join(constants.TACollectorTLSDirPath, f.path),
			SubPath:   f.key,
			ReadOnly:  true,
		})
	}
	return volumes, mounts
}

// mtlsFile describes a single certificate file to project into /tls: which Secret and key it comes
// from, and the fixed filename it must be mounted as.
type mtlsFile struct {
	secretName string
	key        string
	path       string
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
// provide the server and client certificate Secrets; the CA certificate may either be referenced
// separately or bundled in the leaf Secrets under the ca.crt key.
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
	return nil
}
