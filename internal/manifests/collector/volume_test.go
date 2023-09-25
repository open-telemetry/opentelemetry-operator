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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestVolumeNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{}
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
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Volumes: []corev1.Volume{{
				Name: "my-volume",
			}},
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
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			ConfigMaps: []v1alpha1.ConfigMapsSpec{{
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
