// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectJavaagent(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Java
		pod      corev1.Pod
		expected corev1.Pod
		err      error
	}{
		{
			name: "JAVA_TOOL_OPTIONS not defined",
			Java: v1alpha1.Java{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-java",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-java",
							Image:   "foo/bar:1",
							Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation-java/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "test-container",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-java",
									MountPath: "/otel-auto-instrumentation-java-test-container",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/otel-auto-instrumentation-java-test-container/javaagent.jar",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "add extensions to JAVA_TOOL_OPTIONS",
			Java: v1alpha1.Java{Image: "foo/bar:1", Extensions: []v1alpha1.Extensions{
				{Image: "ex/ex:0", Dir: "/ex0"},
				{Image: "ex/ex:1", Dir: "/ex1"},
			}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-java",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-java",
							Image:   "foo/bar:1",
							Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation-java/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-extension-0",
							Image:   "ex/ex:0",
							Command: []string{"cp", "-r", "/ex0/.", "/otel-auto-instrumentation-java/extensions"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-extension-1",
							Image:   "ex/ex:1",
							Command: []string{"cp", "-r", "/ex1/.", "/otel-auto-instrumentation-java/extensions"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "test-container",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-java",
									MountPath: "/otel-auto-instrumentation-java-test-container",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/otel-auto-instrumentation-java-test-container/javaagent.jar -Dotel.javaagent.extensions=/otel-auto-instrumentation-java-test-container/extensions",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "JAVA_TOOL_OPTIONS defined",
			Java: v1alpha1.Java{Image: "foo/bar:1", Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: "-Dbaz=bar",
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
							Name: "opentelemetry-auto-instrumentation-java",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-java",
							Image:   "foo/bar:1",
							Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation-java/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							Name: "test-container",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-java",
									MountPath: "/otel-auto-instrumentation-java-test-container",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: "-Dbaz=bar -javaagent:/otel-auto-instrumentation-java-test-container/javaagent.jar",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "JAVA_TOOL_OPTIONS defined as ValueFrom",
			Java: v1alpha1.Java{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      "JAVA_TOOL_OPTIONS",
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
									Name:      "JAVA_TOOL_OPTIONS",
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envJavaToolsOptions),
		},
		{
			name: "multiple containers with unique volume mount paths",
			Java: v1alpha1.Java{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app1",
						},
						{
							Name: "app2",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-java",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-java",
							Image:   "foo/bar:1",
							Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation-java/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-java",
								MountPath: "/otel-auto-instrumentation-java",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app1",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-java",
									MountPath: "/otel-auto-instrumentation-java-app1",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/otel-auto-instrumentation-java-app1/javaagent.jar",
								},
							},
						},
						{
							Name: "app2",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-java",
									MountPath: "/otel-auto-instrumentation-java-app2",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: " -javaagent:/otel-auto-instrumentation-java-app2/javaagent.jar",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
	}

	injector := sdkInjector{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := test.pod
			var err error
			for i := range pod.Spec.Containers {
				pod, err = injectJavaagent(test.Java, pod, &pod.Spec.Containers[i], v1alpha1.InstrumentationSpec{})
			}
			assert.Equal(t, test.err, err)
			for i := range pod.Spec.Containers {
				injector.injectDefaultJavaEnvVars(&pod.Spec.Containers[i], test.Java)
			}
			assert.Equal(t, test.expected, pod)
		})
	}
}
