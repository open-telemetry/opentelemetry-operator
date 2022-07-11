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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectDotNetSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.DotNet
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name:   "DOTNET_STARTUP_HOOKS, DOTNET_SHARED_STORE, DOTNET_ADDITIONAL_DEPS not defined",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{{Name: "OTEL_DOTNET_TRACER_INSTRUMENTATIONS", Value: "AspNet,HttpClient,SqlClient"}}},
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
									Name:  "OTEL_DOTNET_TRACER_INSTRUMENTATIONS",
									Value: "AspNet,HttpClient,SqlClient",
								},
								{
									Name:  envDotNetStartupHook,
									Value: dotNetStartupHookPath,
								},
								{
									Name:  envDotNetAdditionalDeps,
									Value: dotNetAdditionalDepsPath,
								},
								{
									Name:  envDotNetSharedStore,
									Value: dotNetSharedStorePath,
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "DOTNET_STARTUP_HOOKS, DOTNET_ADDITIONAL_DEPS, DOTNET_SHARED_STORE defined",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetStartupHook,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetAdditionalDeps,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetSharedStore,
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
									Name:  envDotNetStartupHook,
									Value: fmt.Sprintf("%s:%s", "/foo:/bar", dotNetStartupHookPath),
								},
								{
									Name:  envDotNetAdditionalDeps,
									Value: fmt.Sprintf("%s:%s", "/foo:/bar", dotNetAdditionalDepsPath),
								},
								{
									Name:  envDotNetSharedStore,
									Value: fmt.Sprintf("%s:%s", "/foo:/bar", dotNetSharedStorePath),
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "DOTNET_STARTUP_HOOKS defined as ValueFrom",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      envDotNetStartupHook,
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
									Name:      envDotNetStartupHook,
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "DOTNET_ADDITIONAL_DEPS defined as ValueFrom",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      envDotNetAdditionalDeps,
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
									Name:      envDotNetAdditionalDeps,
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "DOTNET_SHARED_STORE defined as ValueFrom",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      envDotNetSharedStore,
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
									Name:      envDotNetSharedStore,
									ValueFrom: &corev1.EnvVarSource{},
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
			pod := injectDotNetSDK(logr.Discard(), test.DotNet, test.pod, 0)
			assert.Equal(t, test.expected, pod)
		})
	}
}
