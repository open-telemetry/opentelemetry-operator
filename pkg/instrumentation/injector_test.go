// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectInjector(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Injector
		pod      corev1.Pod
		expected corev1.Pod
		err      error
	}{
		{
			name:     "LD_PRELOAD not defined",
			Injector: v1alpha1.Injector{Image: "foo/bar:1"},
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
							Name: "opentelemetry-auto-instrumentation-injector",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "opentelemetry-auto-instrumentation-injector",
							Image: "foo/bar:1",
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-injector",
								MountPath: "/otel-auto-instrumentation-injector",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-injector",
									MountPath: "/otel-auto-instrumentation-injector",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_PRELOAD",
									Value: ldPreloadValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:     "LD_PRELOAD defined",
			Injector: v1alpha1.Injector{Image: "foo/bar:1", Resources: testResourceRequirements},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "LD_PRELOAD",
									Value: "-Dbaz=bar",
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
									Name:  "LD_PRELOAD",
									Value: "-Dbaz=bar",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via Value, envVar: %s", envLdPreload),
		},
		{
			name:     "LD_PRELOAD defined as ValueFrom",
			Injector: v1alpha1.Injector{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      "LD_PRELOAD",
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
									Name:      "LD_PRELOAD",
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envLdPreload),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod, err := injectInjector(logr.Discard(), test.Injector, test.pod, 0)
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}
