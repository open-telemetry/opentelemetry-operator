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

type sourceKind string

const (
	sourceSecret    sourceKind = "secret"
	sourceConfigMap sourceKind = "configmap"
)

// mtlsFile is one logical file (ca.crt / tls.crt / tls.key) projected under /tls.
type mtlsFile struct {
	kind sourceKind // secret | configmap
	name string     // Secret or ConfigMap name
	key  string     // source data key, mounted via subPath
	path string     // fixed filename under /tls
}

// volumeIdentity dedups volumes. A Secret and a ConfigMap with the same name must not collide.
type volumeIdentity struct {
	kind sourceKind
	name string
}

func taCertificateVolumes(
	ta *v1alpha1.TargetAllocator,
	volumeName, certManagerSecretName string,
	certRef *v1beta1.CertificateReference,
) ([]corev1.Volume, []corev1.VolumeMount) {
	// cert-manager case: mount the single operator-managed Secret at /tls.
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

	// user-provided case. The CA (required, validated in ValidateTAMTLS) comes from a Secret or a
	// ConfigMap; the leaf certificate and its private key come from (possibly different) Secrets.
	files := make([]mtlsFile, 0, 3)

	if caRef := caCertificateReference(ta.Spec.Mtls); caRef != nil {
		switch {
		case caRef.Secret != nil:
			files = append(files, mtlsFile{
				kind: sourceSecret,
				name: caRef.Secret.Name,
				key:  secretKey(caRef.Secret, constants.TACollectorCAFileName),
				path: constants.TACollectorCAFileName,
			})
		case caRef.ConfigMap != nil:
			files = append(files, mtlsFile{
				kind: sourceConfigMap,
				name: caRef.ConfigMap.Name,
				key:  configMapKey(caRef.ConfigMap, constants.TACollectorCAFileName),
				path: constants.TACollectorCAFileName,
			})
		}
	}

	files = append(files,
		mtlsFile{
			kind: sourceSecret,
			name: certRef.Certificate.Name,
			key:  secretKey(&certRef.Certificate, constants.TACollectorTLSCertFileName),
			path: constants.TACollectorTLSCertFileName,
		},
		mtlsFile{
			kind: sourceSecret,
			name: certRef.Key.Name,
			key:  secretKey(&certRef.Key, constants.TACollectorTLSKeyFileName),
			path: constants.TACollectorTLSKeyFileName,
		},
	)

	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	seen := map[volumeIdentity]string{} // (kind, name) -> volume name
	for _, f := range files {
		id := volumeIdentity{kind: f.kind, name: f.name}
		volName, ok := seen[id]
		if !ok {
			// Derive a stable, unique volume name per distinct source.
			volName = naming.DNSName(naming.Truncate("%s-%d", 63, volumeName, len(volumes)))
			seen[id] = volName
			var source corev1.VolumeSource
			switch f.kind {
			case sourceConfigMap:
				source.ConfigMap = &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: f.name},
				}
			default:
				source.Secret = &corev1.SecretVolumeSource{SecretName: f.name}
			}
			volumes = append(volumes, corev1.Volume{Name: volName, VolumeSource: source})
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

// secretKey returns the selector's data key, or the role-specific default when empty.
func secretKey(sel *v1beta1.SecretKeySelector, def string) string {
	if sel != nil && sel.Key != "" {
		return sel.Key
	}
	return def
}

// configMapKey returns the selector's data key, or the role-specific default when empty.
func configMapKey(sel *v1beta1.ConfigMapKeySelector, def string) string {
	if sel != nil && sel.Key != "" {
		return sel.Key
	}
	return def
}

// ValidateTAMTLS validates the TA mTLS configuration. There are 2 scenarios:
//   - When mTLS relies on cert-manager it requires cert-manager to be available.
//   - When cert-manager is disabled it requires the user to provide the CA
//     reference and both the server and client certificate references.
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
		return errors.New("mTLS is enabled with useCertManager set to false; tls.serverCertificate and tls.clientCertificate must both be set")
	}

	// The CA certificate is required and must reference exactly one of a Secret or a ConfigMap.
	caRef := caCertificateReference(mtls)
	if caRef == nil {
		return errors.New("mTLS is enabled with useCertManager set to false; tls.certificateAuthorityCertificate must be set")
	}
	return validateCAReference(caRef)
}

func validateCAReference(ref *v1beta1.CAReference) error {
	if (ref.Secret != nil) == (ref.ConfigMap != nil) { // neither or both
		return errors.New("tls.certificateAuthorityCertificate must set exactly one of secret or configMap")
	}
	return nil
}
