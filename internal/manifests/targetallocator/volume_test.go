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

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfg "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
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
	assert.Len(t, volumes[0].VolumeSource.ConfigMap.Items, 1)

	// check that it's the ta-internal volume, with the config map
	assert.Equal(t, naming.TAConfigMapVolume(), volumes[0].Name)
}

func TestVolumeWithTargetAllocatorMTLS(t *testing.T) {
	t.Run("CertManager available and EnableTargetAllocatorMTLS enabled", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-targetallocator",
			},
		}
		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

		volumes := Volumes(cfg, ta)

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
		cfg := config.New(config.WithCertManagerAvailability(certmanager.NotAvailable))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

		volumes := Volumes(cfg, ta)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAServerCertificate(ta.Name)})
	})

	t.Run("EnableTargetAllocatorMTLS disabled", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-targetallocator",
			},
		}
		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		volumes := Volumes(cfg, ta)
		assert.NotContains(t, volumes, corev1.Volume{Name: naming.TAServerCertificate(ta.Name)})
	})
}
