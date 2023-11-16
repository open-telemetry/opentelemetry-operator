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

package instrumentation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestInjectGoSDK(t *testing.T) {
	falsee := false
	true := true
	zero := int64(0)

	tests := []struct {
		name string
		v1alpha1.Go
		pod             corev1.Pod
		expected        corev1.Pod
		err             error
		setFeatureGates func(t *testing.T)
	}{
		{
			name: "shared process namespace disabled",
			Go:   v1alpha1.Go{Image: "foo/bar:1", Env: []corev1.EnvVar{}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &falsee,
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &falsee,
				},
			},
			err: fmt.Errorf("shared process namespace has been explicitly disabled"),
		},
		{
			name: "using go-container-names",
			Go:   v1alpha1.Go{Image: "foo/bar:1", Env: []corev1.EnvVar{}},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/go-container-names": "foo,bar",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/go-container-names": "foo,bar",
					},
				},
			},
			err: fmt.Errorf("go instrumentation cannot be injected into a pod, multiple containers configured"),
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
		},
		{
			name: "using container-names",
			Go:   v1alpha1.Go{Image: "foo/bar:1", Env: []corev1.EnvVar{}},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/container-names": "foo,bar",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/container-names": "foo,bar",
					},
				},
			},
			err: fmt.Errorf("go instrumentation cannot be injected into a pod, multiple containers configured"),
		},
		{
			name: "pod annotation takes precedence",
			Go: v1alpha1.Go{
				Image: "foo/bar:1",
				Env: []corev1.EnvVar{
					{
						Name:  "OTEL_GO_AUTO_TARGET_EXE",
						Value: "foo",
					},
				},
				Resources: testResourceRequirements,
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/otel-go-auto-target-exe": "bar",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/otel-go-auto-target-exe": "bar",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name:      sideCarName,
							Resources: testResourceRequirements,
							Image:     "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &true,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_GO_AUTO_TARGET_EXE",
									Value: "bar",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: kernelDebugVolumeName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kernelDebugVolumePath,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "use instrumentation env var",
			Go: v1alpha1.Go{
				Image: "foo/bar:1",
				Env: []corev1.EnvVar{
					{
						Name:  "OTEL_GO_AUTO_TARGET_EXE",
						Value: "foo",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/go-container-names": "foo",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/go-container-names": "foo",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name:  sideCarName,
							Image: "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &true,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_GO_AUTO_TARGET_EXE",
									Value: "foo",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: kernelDebugVolumeName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kernelDebugVolumePath,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "inject env vars",
			Go: v1alpha1.Go{
				Image: "foo/bar:1",
				Env: []corev1.EnvVar{
					{
						Name:  "OTEL_1",
						Value: "foo",
					},
					{
						Name:  "OTEL_2",
						Value: "bar",
					},
				},
			},
			pod: corev1.Pod{},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name:  sideCarName,
							Image: "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &true,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_1",
									Value: "foo",
								},
								{
									Name:  "OTEL_2",
									Value: "bar",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: kernelDebugVolumeName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kernelDebugVolumePath,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setFeatureGates != nil {
				test.setFeatureGates(t)
			}
			pod, err := injectGoSDK(test.Go, test.pod)
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
