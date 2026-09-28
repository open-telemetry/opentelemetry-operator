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

func TestInjectPhpSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Php
		pod              corev1.Pod
		platform         string
		apiVersion       string
		threadSafety     string
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
			platform:     glibc,
			apiVersion:   Php81ApiVersion,
			threadSafety: nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php81ApiVersion, nonZts},
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
			platform:         musl,
			apiVersion:       Php81ApiVersion,
			threadSafety:     nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, musl, Php81ApiVersion, nonZts},
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
			platform:         glibc,
			apiVersion:       Php82ApiVersion,
			threadSafety:     nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php82ApiVersion, nonZts},
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
									Value: phpIniScanDirEnvVarValue,
								},
								{
									Name:  otelPhpAutoloadEnabledrEnvVarName,
									Value: otelPhpAutoloadEnabledrEnvVarValue,
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
							{
								Name:  phpIniScanDirEnvVarName,
								Value: phpIniScanDirEnvVarValue,
							},
							{
								Name:  otelPhpAutoloadEnabledrEnvVarName,
								Value: otelPhpAutoloadEnabledrEnvVarValue,
							},
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
			platform:     glibc,
			apiVersion:   Php81ApiVersion,
			threadSafety: zts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php81ApiVersion, zts},
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
			platform:     glibc,
			apiVersion:   Php83ApiVersion,
			threadSafety: nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php83ApiVersion, nonZts},
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
			platform:     glibc,
			apiVersion:   Php84ApiVersion,
			threadSafety: nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php84ApiVersion, nonZts},
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
			platform:     glibc,
			apiVersion:   Php81ApiVersion,
			threadSafety: nonZts,
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
			platform:     glibc,
			apiVersion:   Php81ApiVersion,
			threadSafety: nonZts,
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
			platform:     glibc,
			apiVersion:   Php85ApiVersion,
			threadSafety: nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php85ApiVersion, nonZts},
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
			platform:     glibc,
			apiVersion:   Php85ApiVersion,
			threadSafety: nonZts,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php85ApiVersion, nonZts},
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
						{
							Name: "my-init",
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
			name: "platform and thread safety default value",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			apiVersion: Php81ApiVersion,
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
					},
					InitContainers: []corev1.Container{
						{
							Name:    "opentelemetry-auto-instrumentation-php",
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, glibc, Php81ApiVersion, nonZts},
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
			name: "apiVersion not defined",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: errors.New("provided instrumentation.opentelemetry.io/otel-php-api-version annotation value '' is not supported"),
		},
		{
			name: "Unsupported platform value",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			platform:   "windows",
			apiVersion: Php81ApiVersion,
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-platform annotation value '%s' is not supported", "windows"),
		},
		{
			name: "Unsupported apiVersion value",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			apiVersion: "invalid",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-api-version annotation value '%s' is not supported", "invalid"),
		},
		{
			name: "Unsupported thread safety value",
			Php:  v1alpha1.Php{Image: "foo/bar:1"},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{}},
				},
			},
			apiVersion:   Php81ApiVersion,
			threadSafety: "invalid",
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			err: fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-thread-safety annotation value '%s' is not supported", "invalid"),
		},
	}

	injector := sdkInjector{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := test.pod

			// Collect all containers (regular first, then init)
			containers := allPhpTestContainers(&pod)

			err := injectPhpSDK(test.Php, &pod, containers, v1alpha1.InstrumentationSpec{}, test.platform, test.apiVersion, test.threadSafety)
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
