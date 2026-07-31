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

// IsTAMTLSEnabled reports whether mTLS should be enabled.
func IsTAMTLSEnabled(mtls *v1beta1.TargetAllocatorMTLS) bool {
	return mtls != nil && mtls.Enabled
}

// IsTAMTLSCertManagerEnabled reports whether cert-manager configuration for mTLS
// is provided + is cert-manager available on the cluster.
func IsTAMTLSCertManagerEnabled(
	mtls *v1beta1.TargetAllocatorMTLS,
	cfg config.Config,
) bool {
	if !IsTAMTLSEnabled(mtls) {
		return false
	}
	if mtls.UseCertManager != nil && !*mtls.UseCertManager {
		return false
	}
	return cfg.CertManagerAvailability == certmanager.Available
}

// IsTAMTLSUserProvided reports whether mTLS is enabled with user-provided
// certificates (i.e. cert-manager is explicitly disabled).
func IsTAMTLSUserProvided(mtls *v1beta1.TargetAllocatorMTLS) bool {
	return IsTAMTLSEnabled(mtls) &&
		mtls.UseCertManager != nil &&
		!*mtls.UseCertManager
}

// TAServerCertificateVolumes builds the volumes and volume mounts that provide
// TA's server certificate (and the CA used to verify collector clients).
// There are 2 scenarios:
//   - With cert-manager it mounts the operator-managed Secret at /tls.
//   - With user-provided certificates it projects each referenced Secret key
//     onto the corresponding file under /tls via subPath mounts.
func TAServerCertificateVolumes(
	ta *v1alpha1.TargetAllocator,
) ([]corev1.Volume, []corev1.VolumeMount) {
	return taCertificateVolumes(
		ta,
		naming.TAServerCertificate(ta.Name),
		naming.TAServerCertificateSecretName(ta.Name),
		taServerCertificateReference(ta.Spec.Mtls),
	)
}

// TAClientCertificateVolumes builds the volumes and volume mounts that provide
// collector's client certificate (and the CA used to verify the TA server).
// There are 2 scenarios:
//   - With cert-manager it mounts the operator-managed Secret at /tls
//   - With user-provided certificates it projects each referenced Secret key
//     onto the corresponding file under /tls via subPath mounts.
func TAClientCertificateVolumes(
	ta *v1alpha1.TargetAllocator,
	otelcolName string,
) ([]corev1.Volume, []corev1.VolumeMount) {
	return taCertificateVolumes(
		ta,
		naming.TAClientCertificate(otelcolName),
		naming.TAClientCertificateSecretName(otelcolName),
		taClientCertificateReference(ta.Spec.Mtls),
	)
}

type mtlsFile struct {
	secretName string
	key        string
	path       string
}

func taCertificateVolumes(
	ta *v1alpha1.TargetAllocator,
	volumeName, certManagerSecretName string,
	certRef *v1beta1.CertificateReference,
) ([]corev1.Volume, []corev1.VolumeMount) {
	// cert-manager case
	if !IsTAMTLSUserProvided(ta.Spec.Mtls) || certRef == nil {
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

	// user-provided case
	// The CA certificate is served from its own reference when provided, otherwise
	// it is expected to be bundled in the leaf Secret under the ca.crt key.
	caRef := caCertificateReference(ta.Spec.Mtls)
	caSecretName := certRef.SecretName
	caKey := constants.TACollectorCAFileName
	if caRef != nil {
		caSecretName = caRef.SecretName
		caKey = dataKeyCA(caRef)
	}

	files := []mtlsFile{
		{
			secretName: caSecretName,
			key:        caKey,
			path:       constants.TACollectorCAFileName,
		},
		{
			secretName: certRef.SecretName,
			key:        dataKeyCertificate(certRef),
			path:       constants.TACollectorTLSCertFileName,
		},
		{
			secretName: certRef.SecretName,
			key:        dataKeyKey(certRef),
			path:       constants.TACollectorTLSKeyFileName,
		},
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

func taTLS(mtls *v1beta1.TargetAllocatorMTLS) *v1beta1.TargetAllocatorTLS {
	if mtls == nil {
		return nil
	}
	return mtls.TLS
}

func taServerCertificateReference(mtls *v1beta1.TargetAllocatorMTLS) *v1beta1.CertificateReference {
	if tls := taTLS(mtls); tls != nil {
		return tls.ServerCertificate
	}
	return nil
}

func taClientCertificateReference(mtls *v1beta1.TargetAllocatorMTLS) *v1beta1.CertificateReference {
	if tls := taTLS(mtls); tls != nil {
		return tls.ClientCertificate
	}
	return nil
}

func caCertificateReference(mtls *v1beta1.TargetAllocatorMTLS) *v1beta1.CAReference {
	if tls := taTLS(mtls); tls != nil {
		return tls.CertificateAuthorityCertificate
	}
	return nil
}

func dataKeyCA(ref *v1beta1.CAReference) string {
	if ref != nil && ref.DataKeyCertificate != "" {
		return ref.DataKeyCertificate
	}
	return constants.TACollectorTLSCertFileName
}

func dataKeyCertificate(ref *v1beta1.CertificateReference) string {
	if ref != nil && ref.DataKeyCertificate != "" {
		return ref.DataKeyCertificate
	}
	return constants.TACollectorTLSCertFileName
}

func dataKeyKey(ref *v1beta1.CertificateReference) string {
	if ref != nil && ref.DataKeyKey != "" {
		return ref.DataKeyKey
	}
	return constants.TACollectorTLSKeyFileName
}

// ValidateTAMTLS validates the TA mTLS configuration. There are 2 scenarios:
//   - When mTLS relies on cert-manager it requires cert-manager to be available.
//   - When cert-manager is disabled it requires the user to provide the server
//     and client certificate Secrets.
//
// Note: The CA certificate may either be referenced separately or bundled in
// the leaf Secrets under the ca.crt key.
func ValidateTAMTLS(mtls *v1beta1.TargetAllocatorMTLS, certManagerAvailable bool) error {
	if !IsTAMTLSEnabled(mtls) {
		return nil
	}

	if !IsTAMTLSUserProvided(mtls) {
		if !certManagerAvailable {
			return errors.New("mTLS is enabled with useCertManager but cert-manager is not available; install cert-manager and restart the operator, or set useCertManager to false")
		}
		return nil
	}

	// User-provided certificates: both leaf certificates must be referenced.
	if taServerCertificateReference(mtls) == nil || taClientCertificateReference(mtls) == nil {
		return errors.New("mTLS is enabled with useCertManager set to false; tls.serverCertificate and tls.clientCertificate must both reference a Secret")
	}
	return nil
}
