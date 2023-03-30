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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectGolangSDK(t *testing.T) {
	falsee := false
	truee := true
	zero := int64(0)

	tests := []struct {
		name string
		v1alpha1.Golang
		pod      corev1.Pod
		expected corev1.Pod
		err      error
	}{
		{
			name:   "shared process namespace disabled",
			Golang: v1alpha1.Golang{Image: "foo/bar:1", Env: []corev1.EnvVar{}},
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
			name:   "using container-names",
			Golang: v1alpha1.Golang{Image: "foo/bar:1", Env: []corev1.EnvVar{}},
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
			err: fmt.Errorf("golang instrumentation cannot be injected into a pod using instrumentation.opentelemetry.io/container-names with more than 1 container"),
		},
		{
			name: "pod annotation takes precedence",
			Golang: v1alpha1.Golang{
				Image: "foo/bar:1",
				Env: []corev1.EnvVar{
					{
						Name:  "OTEL_TARGET_EXE",
						Value: "foo",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/golang-target-exec": "bar",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/golang-target-exec": "bar",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &truee,
					Containers: []corev1.Container{
						{
							Name:  sideCarName,
							Image: "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &truee,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_PTRACE"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_TARGET_EXE",
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
			Golang: v1alpha1.Golang{
				Image: "foo/bar:1",
				Env: []corev1.EnvVar{
					{
						Name:  "OTEL_TARGET_EXE",
						Value: "foo",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/container-names": "foo",
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/container-names": "foo",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &truee,
					Containers: []corev1.Container{
						{
							Name:  sideCarName,
							Image: "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &truee,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_PTRACE"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_TARGET_EXE",
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
			Golang: v1alpha1.Golang{
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
					ShareProcessNamespace: &truee,
					Containers: []corev1.Container{
						{
							Name:  sideCarName,
							Image: "foo/bar:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &truee,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_PTRACE"},
								},
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
			pod, err := injectGolangSDK(test.Golang, test.pod)
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
