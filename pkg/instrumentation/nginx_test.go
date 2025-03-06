// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var nginxSdkInitContainerTestCommand = "echo -e $OTEL_NGINX_I13N_SCRIPT > /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && chmod +x /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && cat /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh \"/opt/opentelemetry-webserver/agent\" \"/opt/opentelemetry-webserver/source-conf\" \"nginx.conf\" \"<<SID-PLACEHOLDER>>\""
var nginxSdkInitContainerTestCommandCustomFile = "echo -e $OTEL_NGINX_I13N_SCRIPT > /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && chmod +x /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && cat /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh && /opt/opentelemetry-webserver/agent/nginx_instrumentation.sh \"/opt/opentelemetry-webserver/agent\" \"/opt/opentelemetry-webserver/source-conf\" \"custom-nginx.conf\" \"<<SID-PLACEHOLDER>>\""
var nginxSdkInitContainerI13nScript = "\nNGINX_AGENT_DIR_FULL=$1\t\\n\nNGINX_AGENT_CONF_DIR_FULL=$2 \\n\nNGINX_CONFIG_FILE=$3 \\n\nNGINX_SID_PLACEHOLDER=$4 \\n\nNGINX_SID_VALUE=$5 \\n\necho \"Input Parameters: $@\" \\n\nset -x \\n\n\\n\ncp -r /opt/opentelemetry/* ${NGINX_AGENT_DIR_FULL} \\n\n\\n\nNGINX_VERSION=$(cat ${NGINX_AGENT_CONF_DIR_FULL}/version.txt) \\n\nNGINX_AGENT_LOG_DIR=$(echo \"${NGINX_AGENT_DIR_FULL}/logs\" | sed 's,/,\\\\/,g') \\n\n\\n\ncat ${NGINX_AGENT_DIR_FULL}/conf/opentelemetry_sdk_log4cxx.xml.template | sed 's,__agent_log_dir__,'${NGINX_AGENT_LOG_DIR}',g'  > ${NGINX_AGENT_DIR_FULL}/conf/opentelemetry_sdk_log4cxx.xml \\n\necho -e $OTEL_NGINX_AGENT_CONF > ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf \\n\nsed -i \"s,${NGINX_SID_PLACEHOLDER},${OTEL_NGINX_SERVICE_INSTANCE_ID},g\" ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf \\n\nsed -i \"1s,^,load_module ${NGINX_AGENT_DIR_FULL}/WebServerModule/Nginx/${NGINX_VERSION}/ngx_http_opentelemetry_module.so;\\\\n,g\" ${NGINX_AGENT_CONF_DIR_FULL}/${NGINX_CONFIG_FILE} \\n\nsed -i \"1s,^,env OTEL_RESOURCE_ATTRIBUTES;\\\\n,g\" ${NGINX_AGENT_CONF_DIR_FULL}/${NGINX_CONFIG_FILE} \\n\nmv ${NGINX_AGENT_CONF_DIR_FULL}/opentelemetry_agent.conf  ${NGINX_AGENT_CONF_DIR_FULL}/conf.d \\n\n\t\t"

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
				ObjectMeta: v1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
			},
			expected: corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
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
							Args:    []string{"cp -r /etc/nginx/* /opt/opentelemetry-webserver/source-conf && export NGINX_VERSION=$( { nginx -v ; } 2>&1 ) && echo ${NGINX_VERSION##*/} > /opt/opentelemetry-webserver/source-conf/version.txt"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelMaxQueueSize 4096;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
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
				ObjectMeta: v1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
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
							Args:    []string{"cp -r /opt/nginx/* /opt/opentelemetry-webserver/source-conf && export NGINX_VERSION=$( { nginx -v ; } 2>&1 ) && echo ${NGINX_VERSION##*/} > /opt/opentelemetry-webserver/source-conf/version.txt"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommandCustomFile},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
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
			}},
		// === Test Removal of probes and lifecycle =============================
		{
			name: "Probes removed on clone init container",
			Nginx: v1alpha1.Nginx{
				Image: "foo/bar:1",
			},
			pod: corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ReadinessProbe: &corev1.Probe{},
							StartupProbe:   &corev1.Probe{},
							LivenessProbe:  &corev1.Probe{},
							Lifecycle:      &corev1.Lifecycle{},
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
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
							Args:    []string{"cp -r /etc/nginx/* /opt/opentelemetry-webserver/source-conf && export NGINX_VERSION=$( { nginx -v ; } 2>&1 ) && echo ${NGINX_VERSION##*/} > /opt/opentelemetry-webserver/source-conf/version.txt"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
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
				ObjectMeta: v1.ObjectMeta{
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
				ObjectMeta: v1.ObjectMeta{
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
							Args:    []string{"cp -r /etc/nginx/* /opt/opentelemetry-webserver/source-conf && export NGINX_VERSION=$( { nginx -v ; } 2>&1 ) && echo ${NGINX_VERSION##*/} > /opt/opentelemetry-webserver/source-conf/version.txt"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace my-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
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
			pod := injectNginxSDK(logr.Discard(), test.Nginx, test.pod, false, 0, "http://otlp-endpoint:4317", resourceMap)
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
				ObjectMeta: v1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
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
							Args:    []string{"cp -r /etc/nginx/* /opt/opentelemetry-webserver/source-conf && export NGINX_VERSION=$( { nginx -v ; } 2>&1 ) && echo ${NGINX_VERSION##*/} > /opt/opentelemetry-webserver/source-conf/version.txt"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nginxAgentConfigVolume,
								MountPath: nginxAgentConfDirFull,
							}},
						},
						{
							Name:    nginxAgentInitContainerName,
							Image:   "foo/bar:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName nginx-service-name;\nNginxModuleServiceNamespace nginx;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
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
			pod := injectNginxSDK(logr.Discard(), test.Nginx, test.pod, false, 0, "http://otlp-endpoint:4317", resourceMap)
			assert.Equal(t, test.expected, pod)
		})
	}
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
