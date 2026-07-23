// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestVolumeNewDefault(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{}
	cfg := config.New()

	volumes := Volumes(cfg, otelcol, nil)

	assert.Len(t, volumes, 1)
	assert.Equal(t, naming.ConfigMapVolume(), volumes[0].Name)
}

func TestVolumeAllowsMoreToBeAdded(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Volumes: []corev1.Volume{{
					Name: "my-volume",
				}},
			},
		},
	}
	cfg := config.New()

	volumes := Volumes(cfg, otelcol, nil)

	assert.Len(t, volumes, 2)
	assert.Equal(t, "my-volume", volumes[1].Name)
}

func TestVolumeWithMoreConfigMaps(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			ConfigMaps: []v1beta1.ConfigMapsSpec{{
				Name:      "configmap-test",
				MountPath: "/",
			}, {
				Name:      "configmap-test2",
				MountPath: "/dir",
			}},
		},
	}
	cfg := config.New()

	volumes := Volumes(cfg, otelcol, nil)

	assert.Len(t, volumes, 3)
	assert.Equal(t, "configmap-configmap-test", volumes[1].Name)
	assert.Equal(t, "configmap-configmap-test2", volumes[2].Name)
}

func TestVolumeWithTargetAllocatorMTLS(t *testing.T) {
	t.Run("CertManager available and TargetAllocator mtls enabled", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-collector",
			},
		}
		cfg := config.Config{
			CertManagerAvailability: certmanager.Available,
		}

		ta := &v1alpha1.TargetAllocator{}
		ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

		volumes := Volumes(cfg, otelcol, ta)

		expectedVolume := corev1.Volume{
			Name: naming.TAClientCertificate(otelcol.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.TAClientCertificateSecretName(otelcol.Name),
				},
			},
		}
		assert.Contains(t, volumes, expectedVolume)
	})

	t.Run("CertManager not available", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{}
		cfg := config.Config{
			CertManagerAvailability: certmanager.NotAvailable,
		}

		ta := &v1alpha1.TargetAllocator{}
		ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

		volumes := Volumes(cfg, otelcol, ta)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAClientCertificate(otelcol.Name)})
	})

	t.Run("TargetAllocator mtls disabled", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{}
		cfg := config.Config{
			CertManagerAvailability: certmanager.Available,
		}

		volumes := Volumes(cfg, otelcol, nil)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAClientCertificate(otelcol.Name)})
	})

	t.Run("Nil TargetAllocator", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{}
		cfg := config.Config{
			CertManagerAvailability: certmanager.Available,
		}

		volumes := Volumes(cfg, otelcol, nil)
		unexpectedVolume := corev1.Volume{
			Name: naming.TAClientCertificate(otelcol.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.TAClientCertificateSecretName(otelcol.Name),
				},
			},
		}
		assert.NotContains(t, volumes, unexpectedVolume)
	})

	t.Run("mTLS with user-provided client certificate secret", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-collector",
			},
		}
		cfg := config.Config{
			CertManagerAvailability: certmanager.NotAvailable,
		}

		ta := &v1alpha1.TargetAllocator{}
		ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
			Enabled:        true,
			UseCertManager: new(false),
			TLS: &v1beta1.TargetAllocatorTLS{
				ServerCertificate: &v1beta1.CertificateReference{SecretName: "my-server-secret"},
				ClientCertificate: &v1beta1.CertificateReference{SecretName: "my-client-secret"},
			},
		}

		volumes := Volumes(cfg, otelcol, ta)

		var found bool
		for _, v := range volumes {
			if v.Secret != nil && v.Secret.SecretName == "my-client-secret" {
				found = true
			}
		}
		assert.True(t, found, "expected a volume backed by the user-provided client secret")
	})
}
