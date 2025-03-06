// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectDotNetSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.DotNet
		pod      corev1.Pod
		runtime  string
		expected corev1.Pod
		err      error
	}{
		{
			name:   "CORECLR_ENABLE_PROFILING, CORECLR_PROFILER, CORECLR_PROFILER_PATH, DOTNET_STARTUP_HOOKS, DOTNET_SHARED_STORE, DOTNET_ADDITIONAL_DEPS, OTEL_DOTNET_AUTO_HOME not defined",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{}, Resources: testResourceRequirements},
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
							Name: "opentelemetry-auto-instrumentation-dotnet",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-dotnet",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-dotnet"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-dotnet",
								MountPath: "/otel-auto-instrumentation-dotnet",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-dotnet",
									MountPath: "/otel-auto-instrumentation-dotnet",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetCoreClrEnableProfiling,
									Value: dotNetCoreClrEnableProfilingEnabled,
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: dotNetCoreClrProfilerID,
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: dotNetCoreClrProfilerGlibcPath,
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
									Name:  envDotNetOTelAutoHome,
									Value: dotNetOTelAutoHomePath,
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
			err: nil,
		},
		{
			name:   "CORECLR_ENABLE_PROFILING, CORECLR_PROFILER, CORECLR_PROFILER_PATH, DOTNET_STARTUP_HOOKS, DOTNET_ADDITIONAL_DEPS, DOTNET_SHARED_STORE defined",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetCoreClrEnableProfiling,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: "/foo:/bar",
								},
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
							Name: "opentelemetry-auto-instrumentation-dotnet",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-dotnet",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-dotnet"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-dotnet",
								MountPath: "/otel-auto-instrumentation-dotnet",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-dotnet",
									MountPath: "/otel-auto-instrumentation-dotnet",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetCoreClrEnableProfiling,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: "/foo:/bar",
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: "/foo:/bar",
								},
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
								{
									Name:  envDotNetOTelAutoHome,
									Value: dotNetOTelAutoHomePath,
								},
							},
						},
					},
				},
			},
			err: nil,
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
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envDotNetStartupHook),
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
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envDotNetAdditionalDeps),
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
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envDotNetSharedStore),
		},
		{
			name:   "OTEL_DOTNET_AUTO_HOME already set in the container",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetOTelAutoHome,
									Value: "/otel-dotnet-auto",
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
									Name:  envDotNetOTelAutoHome,
									Value: "/otel-dotnet-auto",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("OTEL_DOTNET_AUTO_HOME environment variable is already set in the container"),
		},
		{
			name:   "OTEL_DOTNET_AUTO_HOME already set in the .NET instrumentation spec",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{{Name: envDotNetOTelAutoHome, Value: dotNetOTelAutoHomePath}}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: fmt.Errorf("OTEL_DOTNET_AUTO_HOME environment variable is already set in the .NET instrumentation spec"),
		},
		{
			name:   "runtime linux-x64",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{}, Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			runtime: dotNetRuntimeLinuxGlibc,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: dotnetVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    dotnetInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-dotnet"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: "/otel-auto-instrumentation-dotnet",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: "/otel-auto-instrumentation-dotnet",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetCoreClrEnableProfiling,
									Value: dotNetCoreClrEnableProfilingEnabled,
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: dotNetCoreClrProfilerID,
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: dotNetCoreClrProfilerGlibcPath,
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
									Name:  envDotNetOTelAutoHome,
									Value: dotNetOTelAutoHomePath,
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
			err: nil,
		},
		{
			name:   "runtime linux-musl-x64",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{}, Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			runtime: dotNetRuntimeLinuxMusl,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: dotnetVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    dotnetInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-dotnet"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: "/otel-auto-instrumentation-dotnet",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: "/otel-auto-instrumentation-dotnet",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  envDotNetCoreClrEnableProfiling,
									Value: dotNetCoreClrEnableProfilingEnabled,
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: dotNetCoreClrProfilerID,
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: dotNetCoreClrProfilerMuslPath,
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
									Name:  envDotNetOTelAutoHome,
									Value: dotNetOTelAutoHomePath,
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
			err: nil,
		},
		{
			name:   "runtime not-supported",
			DotNet: v1alpha1.DotNet{Image: "foo/bar:1", Env: []corev1.EnvVar{}, Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
				ObjectMeta: metav1.ObjectMeta{},
			},
			runtime: "not-supported",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: fmt.Errorf("provided instrumentation.opentelemetry.io/dotnet-runtime annotation value 'not-supported' is not supported"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod, err := injectDotNetSDK(test.DotNet, test.pod, 0, test.runtime)
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
