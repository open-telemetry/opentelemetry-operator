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

func TestInjectNodeJSSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.NodeJS
		pod      corev1.Pod
		expected corev1.Pod
		err      error
	}{
		{
			name:   "NODE_OPTIONS not defined",
			NodeJS: v1alpha1.NodeJS{Image: "foo/bar:1"},
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
							Name: "opentelemetry-auto-instrumentation-nodejs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-nodejs",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-nodejs"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-nodejs",
								MountPath: "/otel-auto-instrumentation-nodejs",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-nodejs",
									MountPath: "/otel-auto-instrumentation-nodejs",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_OPTIONS",
									Value: " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "NODE_OPTIONS defined",
			NodeJS: v1alpha1.NodeJS{Image: "foo/bar:1", Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_OPTIONS",
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
							Name: "opentelemetry-auto-instrumentation-nodejs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-nodejs",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-nodejs"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-nodejs",
								MountPath: "/otel-auto-instrumentation-nodejs",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-nodejs",
									MountPath: "/otel-auto-instrumentation-nodejs",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_OPTIONS",
									Value: "-Dbaz=bar" + " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "NODE_OPTIONS defined as ValueFrom",
			NodeJS: v1alpha1.NodeJS{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      "NODE_OPTIONS",
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
									Name:      "NODE_OPTIONS",
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envNodeOptions),
		},
		{
			name:   "inject into init container",
			NodeJS: v1alpha1.NodeJS{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "my-init",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-nodejs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-nodejs",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-nodejs"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-nodejs",
								MountPath: "/otel-auto-instrumentation-nodejs",
							}},
						},
						{
							Name: "my-init",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-nodejs",
									MountPath: "/otel-auto-instrumentation-nodejs",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_OPTIONS",
									Value: " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
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

			// Collect all containers (regular first, then init)
			var containers []*corev1.Container
			for i := range pod.Spec.Containers {
				containers = append(containers, &pod.Spec.Containers[i])
			}
			for i := range pod.Spec.InitContainers {
				containers = append(containers, &pod.Spec.InitContainers[i])
			}

			err := injectNodeJSSDK(test.NodeJS, &pod, containers, v1alpha1.InstrumentationSpec{})
			if err != nil {
				assert.Equal(t, test.expected, pod)
				assert.Equal(t, test.err, err)
				return
			}

			for i := range pod.Spec.Containers {
				injector.injectDefaultNodeJSEnvVars(&pod.Spec.Containers[i])
			}
			for i := range pod.Spec.InitContainers {
				// Skip the instrumentation init container we added
				if pod.Spec.InitContainers[i].Name == nodejsInitContainerName {
					continue
				}
				injector.injectDefaultNodeJSEnvVars(&pod.Spec.InitContainers[i])
			}
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
