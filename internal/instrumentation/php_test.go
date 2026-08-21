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

func TestInjectPhpSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Php
		pod              corev1.Pod
		platform         string
		expected         corev1.Pod
		err              error
		inst             v1alpha1.Instrumentation
		simulateDefaults bool
	}{
		{
			name: "PHP_INI_SCAN_DIR not defined",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  phpIniScanDirEnvVarName,
									Value: phpIniScanDirEnvVarValue,
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "spec.env overrides defaults",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			inst:             v1alpha1.Instrumentation{Spec: v1alpha1.InstrumentationSpec{Env: []corev1.EnvVar{{Name: phpIniScanDirEnvVarName, Value: "none"}, {Name: otelPhpAutoloadEnabledrEnvVarName, Value: "false"}}}},
			simulateDefaults: true,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
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
								{
									Name:  phpIniScanDirEnvVarName,
									Value: "none",
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: "false",
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "defaults applied when no spec.env",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			inst:             v1alpha1.Instrumentation{},
			simulateDefaults: true,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{{
						VolumeMounts: []corev1.VolumeMount{{Name: phpVolumeName, MountPath: phpInstrMountPath}},
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
							{Name: phpIniScanDirEnvVarName, Value: phpIniScanDirEnvVarValue},
							{Name: otelPhpAutoloadEnabledrEnvVarName, Value: otelPhpAutoloadEnabledrEnvVarValue},
						},
					}},
				},
			},
			err: nil,
		},
		{
			name: "PHP_INI_SCAN_DIR defined",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "PHP_INI_SCAN_DIR",
									Value: "/dir",
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
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      phpVolumeName,
									MountPath: phpInstrMountPath,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  phpIniScanDirEnvVarName,
									Value: "/dir",
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "OTEL_PHP_AUTOLOAD_ENABLED defined",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_PHP_AUTOLOAD_ENABLED",
									Value: "false",
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
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      phpVolumeName,
									MountPath: phpInstrMountPath,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: "false",
								},
								{
									Name:  phpIniScanDirEnvVarName,
									Value: phpIniScanDirEnvVarValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "OTHER env defined",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTHER",
									Value: "something",
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
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTHER",
									Value: "something",
								},
								{
									Name:  phpIniScanDirEnvVarName,
									Value: phpIniScanDirEnvVarValue,
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "PHP_INI_SCAN_DIR defined as ValueFrom",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      phpIniScanDirEnvVarName,
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
									Name:      phpIniScanDirEnvVarName,
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", phpIniScanDirEnvVarName),
		},
		{
			name: "OTEL_PHP_AUTOLOAD_ENABLED defined as ValueFrom",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:      otelPhpAutoloadEnabledrEnvVarName,
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
									Name:      otelPhpAutoloadEnabledrEnvVarName,
									ValueFrom: &corev1.EnvVarSource{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", otelPhpAutoloadEnabledrEnvVarName),
		},
		{
			name: "OTHER defined as ValueFrom",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name: "OTHER",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
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
							Name: phpVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: phpCloneVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-clone",
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpCloneScript, "--", phpCloneMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "opentelemetry-auto-instrumentation-clone",
								MountPath: phpCloneMountPath,
							}},
						},
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation-clone",
									MountPath: phpCloneMountPath,
								},
								{
									Name:      "opentelemetry-auto-instrumentation-php",
									MountPath: phpInstrMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      phpVolumeName,
									MountPath: phpInstrMountPath,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "OTHER",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  phpIniScanDirEnvVarName,
									Value: phpIniScanDirEnvVarValue,
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "inject into init container",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
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
					InitContainers: []corev1.Container{
						{
							Name: "my-init",
							Env: []corev1.EnvVar{
								{
									Name:  phpIniScanDirEnvVarName,
									Value: phpIniScanDirEnvVarValue,
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
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
			containers := allPhpTestContainers(&pod)

			err := injectPhpSDK(test.Php, &pod, containers, v1alpha1.InstrumentationSpec{})
			if err != nil {
				assert.Equal(t, test.expected, pod)
				assert.Equal(t, test.err, err)
				return
			}

			for i := range pod.Spec.Containers {
				if test.simulateDefaults {
					injector.injectCommonEnvVar(test.inst, &pod.Spec.Containers[i])
				}
				injector.injectDefaultPhpEnvVars(&pod.Spec.Containers[i])
			}
			for i := range pod.Spec.InitContainers {
				// Skip the instrumentation init containers we added
				if pod.Spec.InitContainers[i].Name == phpInitContainerName || pod.Spec.InitContainers[i].Name == phpCloneContainerName {
					continue
				}
				if test.simulateDefaults {
					injector.injectCommonEnvVar(test.inst, &pod.Spec.InitContainers[i])
				}
				injector.injectDefaultPhpEnvVars(&pod.Spec.InitContainers[i])
			}
			assert.Equal(t, test.expected, pod)
			assert.Equal(t, test.err, err)
		})
	}
}

func allPhpTestContainers(pod *corev1.Pod) []*corev1.Container {
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
