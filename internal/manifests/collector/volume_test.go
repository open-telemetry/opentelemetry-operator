// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfg "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestVolumeNewDefault(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{}
	cfg := config.New()

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 1)

	// check that it's the otc-internal volume, with the config map
	assert.Equal(t, naming.ConfigMapVolume(), volumes[0].Name)
}

func TestVolumeAllowsMoreToBeAdded(t *testing.T) {
	// prepare
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

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 2)

	// check that it's the otc-internal volume, with the config map
	assert.Equal(t, "my-volume", volumes[1].Name)
}

func TestVolumeWithMoreConfigMaps(t *testing.T) {
	// prepare
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

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 3)

	// check if the volume with the configmap prefix is mounted after defining the config map.
	assert.Equal(t, "configmap-configmap-test", volumes[1].Name)
	assert.Equal(t, "configmap-configmap-test2", volumes[2].Name)
}

func TestVolumeWithTargetAllocatorMTLS(t *testing.T) {
	t.Run("CertManager available and EnableTargetAllocatorMTLS enabled", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-collector",
			},
		}
		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

		volumes := Volumes(cfg, otelcol)

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
		cfg := config.New(config.WithCertManagerAvailability(certmanager.NotAvailable))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

		volumes := Volumes(cfg, otelcol)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAClientCertificate(otelcol.Name)})
	})

	t.Run("EnableTargetAllocatorMTLS disabled", func(t *testing.T) {
		otelcol := v1beta1.OpenTelemetryCollector{}
		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		volumes := Volumes(cfg, otelcol)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAClientCertificate(otelcol.Name)})
	})
}
