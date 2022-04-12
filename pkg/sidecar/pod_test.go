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

package sidecar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestAddSidecarWhenNoSidecarExists(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
			},
			// cross-test: the pod has a volume already, make sure we don't remove it
			Volumes: []corev1.Volume{{}},
		},
	}
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otelcol-sample",
			Namespace: "some-app",
		},
	}
	cfg := config.New(config.WithCollectorImage("some-default-image"))

	// test
	changed, err := add(cfg, logger, otelcol, pod, nil)

	// verify
	assert.NoError(t, err)
	assert.Len(t, changed.Spec.Containers, 2)
	assert.Len(t, changed.Spec.Volumes, 2)
	assert.Equal(t, "some-app.otelcol-sample", changed.Labels["sidecar.opentelemetry.io/injected"])
}

// this situation should never happen in the current code path, but it should not fail
// if it's asked to add a new sidecar. The caller is expected to have called existsIn before.
func TestAddSidecarWhenOneExistsAlready(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
				{Name: naming.Container()},
			},
		},
	}
	otelcol := v1alpha1.OpenTelemetryCollector{}
	cfg := config.New(config.WithCollectorImage("some-default-image"))

	// test
	changed, err := add(cfg, logger, otelcol, pod, nil)

	// verify
	assert.NoError(t, err)
	assert.Len(t, changed.Spec.Containers, 3)
}

func TestRemoveSidecar(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
				{Name: naming.Container()},
				{Name: naming.Container()}, // two sidecars! should remove both
			},
		},
	}

	// test
	changed, err := remove(pod)

	// verify
	assert.NoError(t, err)
	assert.Len(t, changed.Spec.Containers, 1)
}

func TestRemoveNonExistingSidecar(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
			},
		},
	}

	// test
	changed, err := remove(pod)

	// verify
	assert.NoError(t, err)
	assert.Len(t, changed.Spec.Containers, 1)
}

func TestExistsIn(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		expected bool
		pod      corev1.Pod
	}{
		{"has-sidecar", true, corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "my-app"},
					{Name: naming.Container()},
				},
			},
		}},

		{"does-not-have-sidecar", false, corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, existsIn(tt.pod))
		})
	}
}

func TestAddSidecarWithAditionalEnv(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
			},
		},
	}
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otelcol-sample",
			Namespace: "some-app",
		},
	}
	cfg := config.New(config.WithCollectorImage("some-default-image"))

	extraEnv := corev1.EnvVar{
		Name:  "extraenv",
		Value: "extravalue",
	}

	// test
	changed, err := add(cfg, logger, otelcol, pod, []corev1.EnvVar{
		extraEnv,
	})

	// verify
	assert.NoError(t, err)
	assert.Len(t, changed.Spec.Containers, 2)
	assert.Contains(t, changed.Spec.Containers[1].Env, extraEnv)

}
