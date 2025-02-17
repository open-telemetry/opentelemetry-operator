// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sidecar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var logger = logf.Log.WithName("unit-tests")

func enableSidecarFeatureGate(t *testing.T) {
	originalVal := featuregate.EnableNativeSidecarContainers.IsEnabled()
	t.Logf("original is: %+v", originalVal)
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNativeSidecarContainers.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNativeSidecarContainers.ID(), originalVal))
	})
}

func TestAddNativeSidecar(t *testing.T) {
	enableSidecarFeatureGate(t)
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
			},
			InitContainers: []corev1.Container{
				{
					Name: "my-init",
				},
			},
			// cross-test: the pod has a volume already, make sure we don't remove it
			Volumes: []corev1.Volume{{}},
		},
	}

	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otelcol-native-sidecar",
			Namespace: "some-app",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeSidecar,
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				InitContainers: []corev1.Container{
					{
						Name: "test",
					},
				},
			},
		},
	}

	otelcolYaml, err := otelcol.Spec.Config.Yaml()
	require.NoError(t, err)
	cfg := config.New(config.WithCollectorImage("some-default-image"))

	// test
	changed, err := add(cfg, logger, otelcol, pod, nil)

	// verify
	assert.NoError(t, err)
	require.Len(t, changed.Spec.Containers, 1)
	require.Len(t, changed.Spec.InitContainers, 3)
	require.Len(t, changed.Spec.Volumes, 1)
	assert.Equal(t, "some-app.otelcol-native-sidecar",
		changed.Labels["sidecar.opentelemetry.io/injected"])
	expectedPolicy := corev1.ContainerRestartPolicyAlways
	assert.Equal(t, corev1.Container{
		Name:          "otc-container",
		Image:         "some-default-image",
		Args:          []string{"--config=env:OTEL_CONFIG"},
		RestartPolicy: &expectedPolicy,
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "OTEL_CONFIG",
				Value: string(otelcolYaml),
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "metrics",
				ContainerPort: 8888,
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}, changed.Spec.InitContainers[2])
}

func TestAddSidecarWhenNoSidecarExists(t *testing.T) {
	// prepare
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "my-app"},
			},
			InitContainers: []corev1.Container{
				{
					Name: "my-init",
				},
			},
			// cross-test: the pod has a volume already, make sure we don't remove it
			Volumes: []corev1.Volume{{}},
		},
	}

	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otelcol-sample-with-a-name-that-is-longer-than-sixty-three-characters",
			Namespace: "some-app",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Ports: []v1beta1.PortsSpec{
					{
						ServicePort: corev1.ServicePort{
							Name:     "metrics",
							Port:     8888,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
				InitContainers: []corev1.Container{
					{
						Name: "test",
					},
				},
			},
		},
	}

	otelcolYaml, err := otelcol.Spec.Config.Yaml()
	require.NoError(t, err)
	cfg := config.New(config.WithCollectorImage("some-default-image"))

	// test
	changed, err := add(cfg, logger, otelcol, pod, nil)

	// verify
	assert.NoError(t, err)
	require.Len(t, changed.Spec.Containers, 2)
	require.Len(t, changed.Spec.InitContainers, 2)
	require.Len(t, changed.Spec.Volumes, 1)
	assert.Equal(t, "otelcol-sample-with-a-name-that-is-longer-than-sixty-three-cha",
		changed.Labels["sidecar.opentelemetry.io/injected"])
	assert.Equal(t, corev1.Container{
		Name:  "otc-container",
		Image: "some-default-image",
		Args:  []string{"--config=env:OTEL_CONFIG"},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "OTEL_CONFIG",
				Value: string(otelcolYaml),
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "metrics",
				ContainerPort: 8888,
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}, changed.Spec.Containers[1])
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
	otelcol := v1beta1.OpenTelemetryCollector{}
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
			InitContainers: []corev1.Container{
				{Name: "something"},
				{Name: naming.Container()}, // NOTE: native sidecar since k8s 1.28.
				{Name: naming.Container()}, // two sidecars! should remove both
			},
		},
	}

	// test
	changed := remove(pod)

	// verify
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
	changed := remove(pod)

	// verify
	assert.Len(t, changed.Spec.Containers, 1)
}

func TestExistsIn(t *testing.T) {
	enableSidecarFeatureGate(t)

	for _, tt := range []struct {
		desc     string
		pod      corev1.Pod
		expected bool
	}{
		{"has-sidecar",
			corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
						{Name: naming.Container()},
					},
				},
			},
			true},

		{"does-have-native-sidecar",
			corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
					},
					InitContainers: []corev1.Container{
						{Name: naming.Container()},
					},
				},
			},
			true},

		{"does-not-have-sidecar",
			corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			},
			false},
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
	otelcol := v1beta1.OpenTelemetryCollector{
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
