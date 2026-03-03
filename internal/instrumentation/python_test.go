// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"errors"
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
		pod              corev1.Pod
		platform         string
		expected         corev1.Pod
		err              error
		inst             v1alpha1.Instrumentation
		simulateDefaults bool
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
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "spec.env overrides defaults",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			platform:         "glibc",
			inst:             v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{Env: []corev1.EnvVar{{Name: "OTEL_METRICS_EXPORTER", Value: "none"}}}},
			simulateDefaults: true,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: pythonVolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &defaultVolumeLimitSize},
						},
					}},
					InitContainers: []corev1.Container{{
						Name:         "opentelemetry-auto-instrumentation-python",
						Image:        "foo/bar:1",
						Command:      []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
						VolumeMounts: []corev1.VolumeMount{{Name: pythonVolumeName, MountPath: "/otel-auto-instrumentation-python"}},
					}},
					Containers: []corev1.Container{{
						VolumeMounts: []corev1.VolumeMount{{Name: pythonVolumeName, MountPath: "/otel-auto-instrumentation-python"}},
						Env: []corev1.EnvVar{
							{
								Name: "OTEL_NODE_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
								},
							},
							{
								Name: "OTEL_POD_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"},
								},
							},
							{Name: "PYTHONPATH", Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python")},
							{Name: "OTEL_METRICS_EXPORTER", Value: "none"},
							{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
							{Name: "OTEL_TRACES_EXPORTER", Value: "otlp"},
							{Name: "OTEL_LOGS_EXPORTER", Value: "otlp"},
						},
					}},
				},
			},
			err: nil,
		},
		{
			name:   "defaults applied when no spec.env",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			platform:         "glibc",
			inst:             v1alpha1.Instrumentation{},
			simulateDefaults: true,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: pythonVolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &defaultVolumeLimitSize},
						},
					}},
					InitContainers: []corev1.Container{{
						Name:         "opentelemetry-auto-instrumentation-python",
						Image:        "foo/bar:1",
						Command:      []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
						VolumeMounts: []corev1.VolumeMount{{Name: pythonVolumeName, MountPath: "/otel-auto-instrumentation-python"}},
					}},
					Containers: []corev1.Container{{
						VolumeMounts: []corev1.VolumeMount{{Name: pythonVolumeName, MountPath: "/otel-auto-instrumentation-python"}},
						Env: []corev1.EnvVar{
							{
								Name: "OTEL_NODE_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
								},
							},
							{
								Name: "OTEL_POD_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"},
								},
							},
							{Name: "PYTHONPATH", Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python")},
							{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
							{Name: "OTEL_TRACES_EXPORTER", Value: "otlp"},
							{Name: "OTEL_METRICS_EXPORTER", Value: "otlp"},
							{Name: "OTEL_LOGS_EXPORTER", Value: "otlp"},
						},
					}},
				},
			},
			err: nil,
		},
		{
			name:   "PYTHONPATH defined",
			Python: v1alpha1.Python{Image: "foo/bar:1", Resources: testResourceRequirements},
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
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-python",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/foo:/bar", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
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
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "zipkin",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
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
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-python",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "somebackend",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "OTEL_LOGS_EXPORTER defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_LOGS_EXPORTER",
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
							Name: "opentelemetry-auto-instrumentation-python",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "somebackend",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "OTEL_EXPORTER_OTLP_PROTOCOL defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "somebackend",
								},
							},
						},
					},
				},
			},
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation-python",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "somebackend",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
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
			platform: "glibc",
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
		{
			name:   "musl platform defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			platform: "musl",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation-musl/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "platform not defined",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			platform: "",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name:   "platform not supported",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			platform: "not-supported",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: errors.New("provided instrumentation.opentelemetry.io/otel-python-platform annotation value 'not-supported' is not supported"),
		},
		{
			name:   "inject into init container",
			Python: v1alpha1.Python{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "my-init",
						},
					},
				},
			},
			platform: "glibc",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-python",
							Image:   "foo/bar:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-python",
								MountPath: "/otel-auto-instrumentation-python",
							}},
						},
						{
							Name: "my-init",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-python",
									MountPath: "/otel-auto-instrumentation-python",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation", "/otel-auto-instrumentation-python"),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_LOGS_EXPORTER",
									Value: "otlp",
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
			containers := allContainers(&pod)

			err := injectPythonSDK(test.Python, &pod, containers, test.platform, v1alpha1.InstrumentationSpec{})
			if err != nil {
				assert.Equal(t, test.expected, pod)
				assert.Equal(t, test.err, err)
				return
			}

			for i := range pod.Spec.Containers {
				if test.simulateDefaults {
					injector.injectCommonEnvVar(test.inst, &pod.Spec.Containers[i])
				}
				injector.injectDefaultPythonEnvVars(&pod.Spec.Containers[i])
			}
			for i := range pod.Spec.InitContainers {
				// Skip the instrumentation init container we added
				if pod.Spec.InitContainers[i].Name == pythonInitContainerName {
					continue
				}
				if test.simulateDefaults {
					injector.injectCommonEnvVar(test.inst, &pod.Spec.InitContainers[i])
				}
				injector.injectDefaultPythonEnvVars(&pod.Spec.InitContainers[i])
			}
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}

func allContainers(pod *corev1.Pod) []*corev1.Container {
	// Collect all containers (regular first, then init)
	var containers []*corev1.Container
	for i := range pod.Spec.Containers {
		containers = append(containers, &pod.Spec.Containers[i])
	}
	for i := range pod.Spec.InitContainers {
		containers = append(containers, &pod.Spec.InitContainers[i])
	}
	return containers
}
