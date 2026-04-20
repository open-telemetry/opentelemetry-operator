// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestVolumeNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.TargetAllocator{}
	cfg := config.New()

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 1)

	// check if the number of elements in the volume source items list is 1
	assert.Len(t, volumes[0].ConfigMap.Items, 1)

	// check that it's the ta-internal volume, with the config map
	assert.Equal(t, naming.TAConfigMapVolume(), volumes[0].Name)
}

func TestUserDefinedVolume(t *testing.T) {
	ta := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-targetallocator",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Volumes: []corev1.Volume{
					{
						Name: "custom-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}
	cfg := config.New()

	volumes := Volumes(cfg, ta)

	assert.Len(t, volumes, 2)
	assert.Contains(t, volumes, ta.Spec.Volumes[0])
}

func TestVolumeWithTargetAllocatorMTLS(t *testing.T) {
	t.Run("CertManager available and targetAllocator mtls enabled", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-targetallocator",
			},
		}
		collector := &v1beta1.OpenTelemetryCollector{
			Spec: v1beta1.OpenTelemetryCollectorSpec{
				TargetAllocator: v1beta1.TargetAllocatorEmbedded{
					Enabled: true,
					Mtls:    &v1beta1.TargetAllocatorMTLS{Enabled: true},
				},
			},
		}
		cfg := config.Config{
			CertManagerAvailability: certmanager.Available,
		}

		volumes := Volumes(cfg, ta, collector)

		expectedVolume := corev1.Volume{
			Name: naming.TAServerCertificate(ta.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.TAServerCertificateSecretName(ta.Name),
				},
			},
		}
		assert.Contains(t, volumes, expectedVolume)
	})

	t.Run("CertManager not available", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-targetallocator",
			},
		}
		collector := &v1beta1.OpenTelemetryCollector{
			Spec: v1beta1.OpenTelemetryCollectorSpec{
				TargetAllocator: v1beta1.TargetAllocatorEmbedded{
					Enabled: true,
					Mtls:    &v1beta1.TargetAllocatorMTLS{Enabled: true},
				},
			},
		}
		cfg := config.Config{
			CertManagerAvailability: certmanager.NotAvailable,
		}

		volumes := Volumes(cfg, ta, collector)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAServerCertificate(ta.Name)})
	})

	t.Run("targetAllocator mtls disabled", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-targetallocator",
			},
		}
		cfg := config.Config{
			CertManagerAvailability: certmanager.Available,
		}

		volumes := Volumes(cfg, ta, nil)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAServerCertificate(ta.Name)})
	})
}
