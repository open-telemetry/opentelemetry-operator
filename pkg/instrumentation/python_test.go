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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectPythonSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Python
		pod      corev1.Pod
		expected corev1.Pod
		err      error
	}{
		{
			name:   "PYTHONPATH not defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/otel-auto-instrumentation",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp_proto_http",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp_proto_http",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "PYTHONPATH defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: "/foo:/bar",
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/otel-auto-instrumentation",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s:%s", pythonPathPrefix, "/foo:/bar", pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp_proto_http",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp_proto_http",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "OTEL_TRACES_EXPORTER defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "zipkin",
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/otel-auto-instrumentation",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "zipkin",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp_proto_http",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "OTEL_METRICS_EXPORTER defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "somebackend",
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: "/otel-auto-instrumentation",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "somebackend",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp_proto_http",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "PYTHONPATH defined as ValueFrom",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      "PYTHONPATH",
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      "PYTHONPATH",
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envPythonPath),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod, err := injectPythonSDK(test.Python, test.pod, 0)
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
