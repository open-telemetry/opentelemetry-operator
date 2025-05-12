// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestExporter(t *testing.T) {
	tests := []struct {
		name     string
		exporter v1alpha1.Exporter
		expected corev1.Pod
	}{
		{
			name: "ca, crt and key from secret",
			exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
				TLS: &v1alpha1.TLS{
					SecretName: "my-certs",
					CA:         "ca.crt",
					Cert:       "cert.crt",
					Key:        "key.key",
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "otel-auto-secret-my-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-certs",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "otel-auto-secret-my-certs",
									ReadOnly:  true,
									MountPath: "/otel-auto-instrumentation-secret-my-certs",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "https://collector:4318",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CERTIFICATE",
									Value: "/otel-auto-instrumentation-secret-my-certs/ca.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
									Value: "/otel-auto-instrumentation-secret-my-certs/cert.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_KEY",
									Value: "/otel-auto-instrumentation-secret-my-certs/key.key",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "crt and key from secret and ca from configmap",
			exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
				TLS: &v1alpha1.TLS{
					SecretName:    "my-certs",
					ConfigMapName: "ca-bundle",
					CA:            "ca.crt",
					Cert:          "cert.crt",
					Key:           "key.key",
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "otel-auto-secret-my-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-certs",
								},
							},
						},
						{
							Name: "otel-auto-configmap-ca-bundle",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "ca-bundle",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "otel-auto-secret-my-certs",
									ReadOnly:  true,
									MountPath: "/otel-auto-instrumentation-secret-my-certs",
								},
								{
									Name:      "otel-auto-configmap-ca-bundle",
									ReadOnly:  true,
									MountPath: "/otel-auto-instrumentation-configmap-ca-bundle",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "https://collector:4318",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CERTIFICATE",
									Value: "/otel-auto-instrumentation-configmap-ca-bundle/ca.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
									Value: "/otel-auto-instrumentation-secret-my-certs/cert.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_KEY",
									Value: "/otel-auto-instrumentation-secret-my-certs/key.key",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "ca, crt key absolute paths",
			exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
				TLS: &v1alpha1.TLS{
					CA:   "/ca.crt",
					Cert: "/cert.crt",
					Key:  "/key.key",
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "https://collector:4318",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CERTIFICATE",
									Value: "/ca.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
									Value: "/cert.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_KEY",
									Value: "/key.key",
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
			pod := corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			}
			configureExporter(test.exporter, &pod, &pod.Spec.Containers[0])
			assert.Equal(t, test.expected, pod)
		})
	}
}
