// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var (
	nginxSdkInitContainerTestArgs           = []string{nginxAgentScript, "--", "nginx.conf"}
	nginxSdkInitContainerTestArgsCustomFile = []string{nginxAgentScript, "--", "custom-nginx.conf"}
)

func TestInjectNginxSDK(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Nginx
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name: "Clone Container not present",
			Nginx: v1alpha1.Nginx{
				Image: "foo/bar:1",
				Attrs: []corev1.EnvVar{
					{
						Name:  "NginxModuleOtelMaxQueueSize",
						Value: "4096",
					},
				},
			},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nginxAgentConfigVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: nginxAgentVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nginxAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxCloneScript, "--", "/etc/nginx"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    nginxSdkInitContainerTestArgs,
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelMaxQueueSize 4096;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name: nginxServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: nginxAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: "/etc/nginx",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
							},
						},
					},
				},
			},
		},
		// === Test ConfigFile configuration =============================
		{
			name: "ConfigFile honored",
			Nginx: v1alpha1.Nginx{
				Image:      "foo/bar:1",
				ConfigFile: "/opt/nginx/custom-nginx.conf",
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nginxAgentConfigVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: nginxAgentVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nginxAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxCloneScript, "--", "/opt/nginx"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    nginxSdkInitContainerTestArgsCustomFile,
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name: nginxServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: nginxAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: "/opt/nginx",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
							},
						},
					},
				},
			},
		},
		// === Test init-container-incompatible fields not copied =============================
		{
			name: "Init-container-incompatible fields not copied",
			Nginx: v1alpha1.Nginx{
				Image: "foo/bar:1",
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ReadinessProbe: &corev1.Probe{},
							StartupProbe:   &corev1.Probe{},
							LivenessProbe:  &corev1.Probe{},
							Lifecycle:      &corev1.Lifecycle{},
							ResizePolicy: []corev1.ContainerResizePolicy{
								{ResourceName: corev1.ResourceCPU, RestartPolicy: corev1.NotRequired},
								{ResourceName: corev1.ResourceMemory, RestartPolicy: corev1.NotRequired},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nginxAgentConfigVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: nginxAgentVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nginxAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxCloneScript, "--", "/etc/nginx"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    nginxSdkInitContainerTestArgs,
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name: nginxServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: nginxAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: "/etc/nginx",
								},
							},
							ReadinessProbe: &corev1.Probe{},
							StartupProbe:   &corev1.Probe{},
							LivenessProbe:  &corev1.Probe{},
							Lifecycle:      &corev1.Lifecycle{},
							ResizePolicy: []corev1.ContainerResizePolicy{
								{ResourceName: corev1.ResourceCPU, RestartPolicy: corev1.NotRequired},
								{ResourceName: corev1.ResourceMemory, RestartPolicy: corev1.NotRequired},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
							},
						},
					},
				},
			},
		},
		// Pod Namespace specified
		{
			name:  "Pod Namespace specified",
			Nginx: v1alpha1.Nginx{Image: "foo/bar:1"},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nginxAgentConfigVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: nginxAgentVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nginxAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxCloneScript, "--", "/etc/nginx"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    nginxSdkInitContainerTestArgs,
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace my-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name: nginxServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: nginxAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: "/etc/nginx",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
							},
						},
					},
				},
			},
		},
	}

	resourceMap := map[string]string{
		string(semconv.K8SDeploymentNameKey): "nginx-service-name",
		string(semconv.K8SNamespaceNameKey):  "req-namespace",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := injectNginxSDK(logr.Discard(), test.Nginx, test.pod, false, &test.pod.Spec.Containers[0], "http://otlp-endpoint:4317", resourceMap, v1alpha1.InstrumentationSpec{})
			assert.Equal(t, test.expected, pod)
		})
	}
}

func TestInjectNginxUnknownNamespace(t *testing.T) {
	tests := []struct {
		name string
		v1alpha1.Nginx
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name:  "Clone Container not present, unknown namespace",
			Nginx: v1alpha1.Nginx{Image: "foo/bar:1"},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nginxAgentConfigVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: nginxAgentVolume,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nginxAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxCloneScript, "--", "/etc/nginx"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    nginxSdkInitContainerTestArgs,
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace nginx;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name: nginxServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: nginxAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nginxAgentVolume,
									MountPath: nginxAgentDirFull,
								},
								{
									Name:      nginxAgentConfigVolume,
									MountPath: "/etc/nginx",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
							},
						},
					},
				},
			},
		},
	}

	resourceMap := map[string]string{
		string(semconv.K8SDeploymentNameKey): "nginx-service-name",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := injectNginxSDK(logr.Discard(), test.Nginx, test.pod, false, &test.pod.Spec.Containers[0], "http://otlp-endpoint:4317", resourceMap, v1alpha1.InstrumentationSpec{})
			assert.Equal(t, test.expected, pod)
		})
	}
}

// Regression test: append aliasing corrupts clone init container mounts when
// the VolumeMounts slice has spare capacity.
func TestInjectNginxSDKVolumemountAliasing(t *testing.T) {
	mounts := make([]corev1.VolumeMount, 0, 4)
	mounts = append(mounts,
		corev1.VolumeMount{Name: "a", MountPath: "/a"},
		corev1.VolumeMount{Name: "b", MountPath: "/b"},
	)

	pod := corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{VolumeMounts: mounts}}}}
	result := injectNginxSDK(logr.Discard(), v1alpha1.Nginx{Image: "foo/bar:1"},
		pod, false, &pod.Spec.Containers[0], "http://otlp:4317",
		map[string]string{string(semconv.K8SDeploymentNameKey): "svc"}, v1alpha1.InstrumentationSpec{})

	clone := result.Spec.InitContainers[0]
	lastMount := clone.VolumeMounts[len(clone.VolumeMounts)-1]
	assert.Equal(t, nginxAgentConfigVolume, lastMount.Name,
		"clone's config mount was corrupted by slice aliasing: %v", clone.VolumeMounts)
	assert.Equal(t, nginxAgentConfDirFull, lastMount.MountPath)
}

func TestNginxInitContainerMissing(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{
			name: "InitContainer_Already_Inject",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "istio-init",
						},
						{
							Name: nginxAgentInitContainerName,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InitContainer_Absent_1",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "istio-init",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "InitContainer_Absent_2",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isNginxInitContainerMissing(test.pod, nginxAgentInitContainerName)
			assert.Equal(t, test.expected, result)
		})
	}
}
