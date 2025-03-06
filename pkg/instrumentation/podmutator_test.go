// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestMutatePod(t *testing.T) {

	true := true
	zero := int64(0)

	tests := []struct {
		name            string
		err             string
		pod             corev1.Pod
		expected        corev1.Pod
		inst            v1alpha1.Instrumentation
		ns              corev1.Namespace
		secret          *corev1.Secret
		configMap       *corev1.ConfigMap
		setFeatureGates func(t *testing.T)
		config          config.Config
	}{
		{
			name: "javaagent injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "javaagent",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-certs",
					Namespace: "javaagent",
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-ca-bundle",
					Namespace: "javaagent",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "javaagent",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_JAVAAGENT_DEBUG",
								Value: "true",
							},
							{
								Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
								Value: "false",
							},
							{
								Name:  "SPLUNK_PROFILER_ENABLED",
								Value: "false",
							},
						},
						Resources: testResourceRequirements,
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
						TLS: &v1alpha1.TLS{
							SecretName:    "my-certs",
							ConfigMapName: "my-ca-bundle",
							CA:            "ca.crt",
							Cert:          "cert.crt",
							Key:           "key.key",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: javaVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: "otel-auto-secret-my-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-certs",
								},
							},
						},
						{
							Name: "otel-auto-configmap-my-ca-bundle",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-ca-bundle",
									},
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    javaInitContainerName,
							Command: []string{"cp", "/javaagent.jar", javaInstrMountPath + "/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      javaVolumeName,
								MountPath: javaInstrMountPath,
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_JAVAAGENT_DEBUG",
									Value: "true",
								},
								{
									Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
									Value: "false",
								},
								{
									Name:  "SPLUNK_PROFILER_ENABLED",
									Value: "false",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaAgent,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CERTIFICATE",
									Value: "/otel-auto-instrumentation-configmap-my-ca-bundle/ca.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
									Value: "/otel-auto-instrumentation-secret-my-certs/cert.crt",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_CLIENT_KEY",
									Value: "/otel-auto-instrumentation-secret-my-certs/key.key",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=javaagent,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=javaagent.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
								{
									Name:      "otel-auto-secret-my-certs",
									ReadOnly:  true,
									MountPath: "/otel-auto-instrumentation-secret-my-certs",
								},
								{
									Name:      "otel-auto-configmap-my-ca-bundle",
									ReadOnly:  true,
									MountPath: "/otel-auto-instrumentation-configmap-my-ca-bundle",
								},
							},
						},
					},
				},
			},
			config: config.New(),
		},
		{
			name: "javaagent injection multiple containers, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "javaagent-multiple-containers",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "javaagent-multiple-containers",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_JAVAAGENT_DEBUG",
								Value: "true",
							},
							{
								Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
								Value: "false",
							},
							{
								Name:  "SPLUNK_PROFILER_ENABLED",
								Value: "false",
							},
						},
						Resources: testResourceRequirements,
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava:          "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava:          "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: javaVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    javaInitContainerName,
							Command: []string{"cp", "/javaagent.jar", javaInstrMountPath + "/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      javaVolumeName,
								MountPath: javaInstrMountPath,
							}},
							Resources: testResourceRequirements,
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_JAVAAGENT_DEBUG",
									Value: "true",
								},
								{
									Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
									Value: "false",
								},
								{
									Name:  "SPLUNK_PROFILER_ENABLED",
									Value: "false",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaAgent,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app1",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app1,k8s.namespace.name=javaagent-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=javaagent-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
							},
						},
						{
							Name: "app2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_JAVAAGENT_DEBUG",
									Value: "true",
								},
								{
									Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
									Value: "false",
								},
								{
									Name:  "SPLUNK_PROFILER_ENABLED",
									Value: "false",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaAgent,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app2",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app2,k8s.namespace.name=javaagent-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=javaagent-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(),
		},
		{
			name: "javaagent injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "javaagent-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "javaagent-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_JAVAAGENT_DEBUG",
								Value: "true",
							},
							{
								Name:  "OTEL_INSTRUMENTATION_JDBC_ENABLED",
								Value: "false",
							},
							{
								Name:  "SPLUNK_PROFILER_ENABLED",
								Value: "false",
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			config: config.New(config.WithEnableJavaInstrumentation(false)),
		},
		{
			name: "nodejs injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "nodejs",
				},
				Spec: v1alpha1.InstrumentationSpec{
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_NODEJS_DEBUG",
								Value: "true",
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS: "true",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nodejsVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nodejsInitContainerName,
							Image:   "otel/nodejs:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nodejsVolumeName,
								MountPath: nodejsInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_NODEJS_DEBUG",
									Value: "true",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=nodejs,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=nodejs.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableNodeJSInstrumentation(true)),
		},
		{
			name: "nodejs injection multiple containers, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs-multiple-containers",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "nodejs-multiple-containers",
				},
				Spec: v1alpha1.InstrumentationSpec{
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_NODEJS_DEBUG",
								Value: "true",
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: nodejsVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    nodejsInitContainerName,
							Image:   "otel/nodejs:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nodejsVolumeName,
								MountPath: nodejsInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_NODEJS_DEBUG",
									Value: "true",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app1",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app1,k8s.namespace.name=nodejs-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=nodejs-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
						{
							Name: "app2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_NODEJS_DEBUG",
									Value: "true",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app2",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app2,k8s.namespace.name=nodejs-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=nodejs-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableNodeJSInstrumentation(true)),
		},
		{
			name: "nodejs injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nodejs-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "nodejs-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_NODEJS_DEBUG",
								Value: "true",
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "http://localhost:4317",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectNodeJS: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
		},
		{
			name: "python injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "python",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "python",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
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
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4318",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
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
							Name:    pythonInitContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", pythonInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      pythonVolumeName,
								MountPath: pythonInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
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
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=python,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=python.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pythonVolumeName,
									MountPath: pythonInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnablePythonInstrumentation(true)),
		},
		{
			name: "python injection multiple containers, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "python-multiple-containers",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "python-multiple-containers",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
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
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4318",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
							Name:    pythonInitContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", pythonInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      pythonVolumeName,
								MountPath: pythonInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
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
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app1",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app1,k8s.namespace.name=python-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=python-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pythonVolumeName,
									MountPath: pythonInstrMountPath,
								},
							},
						},
						{
							Name: "app2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
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
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app2",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app2,k8s.namespace.name=python-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=python-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pythonVolumeName,
									MountPath: pythonInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnablePythonInstrumentation(true)),
		},
		{
			name: "python injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "python-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "python-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
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
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4318",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
		},
		{
			name: "dotnet injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dotnet",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "dotnet",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:  "true",
						annotationDotNetRuntime: dotNetRuntimeLinuxMusl,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:  "true",
						annotationDotNetRuntime: dotNetRuntimeLinuxMusl,
					},
				},
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
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: dotnetInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
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
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=dotnet,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=dotnet.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableDotNetInstrumentation(true)),
		},
		{
			name: "dotnet injection, by namespace annotations",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dotnet-by-namespace-annotation",
					Annotations: map[string]string{
						annotationInjectDotNet:  "example-inst",
						annotationDotNetRuntime: dotNetRuntimeLinuxMusl,
					},
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "dotnet-by-namespace-annotation",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{},
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
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: dotnetInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
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
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=dotnet-by-namespace-annotation,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=dotnet-by-namespace-annotation.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableDotNetInstrumentation(true)),
		},
		{
			name: "dotnet injection multiple containers, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dotnet-multiple-containers",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "dotnet-multiple-containers",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:        "true",
						annotationInjectContainerName: "app1,app2",
					},
				},
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
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: dotnetInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
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
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app1",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app1,k8s.namespace.name=dotnet-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=dotnet-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
						{
							Name: "app2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
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
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app2",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app2,k8s.namespace.name=dotnet-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=dotnet-multiple-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableDotNetInstrumentation(true)),
		},
		{
			name: "dotnet injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dotnet-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "dotnet-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			config: config.New(config.WithEnableDotNetInstrumentation(false)),
		},
		{
			name: "go injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "go",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "go",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Go: v1alpha1.Go{
						Image: "otel/go:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectGo:   "true",
						annotationGoExecPath: "/app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectGo:   "true",
						annotationGoExecPath: "/app",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name: "app",
						},
						{
							Name:  sideCarName,
							Image: "otel/go:1",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &zero,
								Privileged: &true,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/kernel/debug",
									Name:      kernelDebugVolumeName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_GO_AUTO_TARGET_EXE",
									Value: "/app",
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
									Value: "20",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.85",
								},
								{
									Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
									Value: "true",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=go,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=go.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: kernelDebugVolumeName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kernelDebugVolumePath,
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableGoInstrumentation(true)),
		},
		{
			name: "go injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "go-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "go-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Go: v1alpha1.Go{
						Image: "otel/go:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
						{
							Name:  "SPLUNK_TRACE_RESPONSE_HEADER_ENABLED",
							Value: "true",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectGo:   "true",
						annotationGoExecPath: "/app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectGo:   "true",
						annotationGoExecPath: "/app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			config: config.New(config.WithEnableGoInstrumentation(false)),
		},
		{
			name: "apache httpd injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "apache-httpd",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "apache-httpd",
				},
				Spec: v1alpha1.InstrumentationSpec{
					ApacheHttpd: v1alpha1.ApacheHttpd{
						Image: "otel/apache-httpd:1",
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectApacheHttpd: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectApacheHttpd: "true",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "otel-apache-conf-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: "otel-apache-agent",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    apacheAgentCloneContainerName,
							Image:   "",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"cp -r /usr/local/apache2/conf/* " + apacheAgentDirectory + apacheAgentConfigDirectory},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      apacheAgentConfigVolume,
								MountPath: apacheAgentDirectory + apacheAgentConfigDirectory,
							}},
						},
						{
							Name:    apacheAgentInitContainerName,
							Image:   "otel/apache-httpd:1",
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								"cp -r /opt/opentelemetry/* /opt/opentelemetry-webserver/agent && export agentLogDir=$(echo \"/opt/opentelemetry-webserver/agent/logs\" | sed 's,/,\\\\/,g') && cat /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml &&echo \"$OTEL_APACHE_AGENT_CONF\" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && sed -i 's/<<SID-PLACEHOLDER>>/'${APACHE_SERVICE_INSTANCE_ID}'/g' /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && echo -e '\nInclude /usr/local/apache2/conf/opentemetry_agent.conf' >> /opt/opentelemetry-webserver/source-conf/httpd.conf"},
							Env: []corev1.EnvVar{
								{
									Name:  apacheAttributesEnvVar,
									Value: "\n#Load the Otel Webserver SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_common.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_resources.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_trace.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_otlp_recordable.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_ostream_span.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_otlp_grpc.so\n#Load the Otel ApacheModule SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_webserver_sdk.so\n#Load the Apache Module. In this example for Apache 2.4\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Load the Apache Module. In this example for Apache 2.2\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel22.so\nLoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Attributes\nApacheModuleEnabled ON\nApacheModuleOtelExporterEndpoint http://collector:12345\nApacheModuleOtelSpanExporter otlp\nApacheModuleResolveBackends  ON\nApacheModuleServiceInstanceId <<SID-PLACEHOLDER>>\nApacheModuleServiceName app\nApacheModuleServiceNamespace apache-httpd\nApacheModuleTraceAsError  ON\n",
								},
								{
									Name: apacheServiceInstanceIdEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      apacheAgentVolume,
									MountPath: apacheAgentDirectory + apacheAgentSubDirectory,
								},
								{
									Name:      apacheAgentConfigVolume,
									MountPath: apacheAgentConfDirFull,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      apacheAgentVolume,
									MountPath: apacheAgentDirectory + apacheAgentSubDirectory,
								},
								{
									Name:      apacheAgentConfigVolume,
									MountPath: apacheDefaultConfigDirectory,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=app,k8s.namespace.name=apache-httpd,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=apache-httpd.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableApacheHttpdInstrumentation(true)),
		},
		{
			name: "apache httpd injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "apache-httpd-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "apache-httpd-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					ApacheHttpd: v1alpha1.ApacheHttpd{
						Image: "otel/apache-httpd:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectApacheHttpd: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectApacheHttpd: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			config: config.New(config.WithEnableApacheHttpdInstrumentation(false)),
		},

		{
			name: "nginx injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "req-namespace",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-nginx-6c44bcbdd",
					Namespace: "req-namespace",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Nginx: v1alpha1.Nginx{
						Image: "otel/nginx-inj:1",
						Attrs: []corev1.EnvVar{{
							Name:  "NginxModuleOtelMaxQueueSize",
							Value: "4096",
						}},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://otlp-endpoint:4317",
					},
					Env: []corev1.EnvVar{},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
					Annotations: map[string]string{
						annotationInjectNginx: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
					Annotations: map[string]string{
						annotationInjectNginx: "true",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "otel-nginx-conf-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "otel-nginx-agent",
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
							Image:   "otel/nginx-inj:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelMaxQueueSize 4096;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName my-nginx-6c44bcbdd;\nNginxModuleServiceNamespace req-namespace;\nNginxModuleTraceAsError ON;\n",
								},
								{
									Name:  "OTEL_NGINX_I13N_SCRIPT",
									Value: nginxSdkInitContainerI13nScript,
								}, {
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
							Name: "nginx",
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
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "LD_LIBRARY_PATH",
									Value: "/opt/opentelemetry-webserver/agent/sdk_lib/lib",
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "my-nginx-6c44bcbdd",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://otlp-endpoint:4317",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=nginx,k8s.namespace.name=req-namespace,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=req-namespace.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).nginx",
								},
							},
						},
					},
				},
			},
			config: config.New(config.WithEnableNginxInstrumentation(true)),
		},
		{
			name: "nginx injection feature gate disabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx-disabled",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-nginx-6c44bcbdd",
					Namespace: "nginx-disabled",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Nginx: v1alpha1.Nginx{
						Image: "otel/nginx-inj:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
							{
								Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
								Value: "http://localhost:4317",
							},
						},
						Attrs: []corev1.EnvVar{{
							Name:  "NginxModuleOtelMaxQueueSize",
							Value: "4096",
						}},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://otlp-endpoint:4317",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_EXPORTER_OTLP_TIMEOUT",
							Value: "20",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER",
							Value: "parentbased_traceidratio",
						},
						{
							Name:  "OTEL_TRACES_SAMPLER_ARG",
							Value: "0.85",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
					Annotations: map[string]string{
						annotationInjectNginx: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
					Annotations: map[string]string{
						annotationInjectNginx: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx",
						},
					},
				},
			},
			config: config.New(config.WithEnableDotNetInstrumentation(false)),
		},

		{
			name: "missing annotation",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-annotation",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "missing-annotation",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Image: "otel/java:1",
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
		},
		{
			name: "annotation set to false",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "annotation-false",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "annotation-false",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Image: "otel/java:1",
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
		},
		{
			name: "annotation set to non existing instance",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-existing-instance",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "non-existing-instance",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{
						Image: "otel/java:1",
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "doesnotexists",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			err: `instrumentations.opentelemetry.io "doesnotexists" not found`,
		},
		{
			name: "multi instrumentation for multiple containers feature gate enabled",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-multi-containers",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-multi-containers",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Java: v1alpha1.Java{
						Image: "otel/java:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectJava:                 "true",
						annotationInjectNodeJS:               "true",
						annotationInjectPython:               "true",
						annotationInjectDotnetContainersName: "dotnet1,dotnet2",
						annotationInjectJavaContainersName:   "java1,java2",
						annotationInjectNodeJSContainersName: "nodejs1,nodejs2",
						annotationInjectPythonContainersName: "python1,python2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "dotnet2",
						},
						{
							Name: "java1",
						},
						{
							Name: "java2",
						},
						{
							Name: "nodejs1",
						},
						{
							Name: "nodejs2",
						},
						{
							Name: "python1",
						},
						{
							Name: "python2",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectJava:                 "true",
						annotationInjectNodeJS:               "true",
						annotationInjectPython:               "true",
						annotationInjectDotnetContainersName: "dotnet1,dotnet2",
						annotationInjectJavaContainersName:   "java1,java2",
						annotationInjectNodeJSContainersName: "nodejs1,nodejs2",
						annotationInjectPythonContainersName: "python1,python2",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: javaVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: nodejsVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
						{
							Name: pythonVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									SizeLimit: &defaultVolumeLimitSize,
								},
							},
						},
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
							Name:    javaInitContainerName,
							Image:   "otel/java:1",
							Command: []string{"cp", "/javaagent.jar", javaInstrMountPath + "/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      javaVolumeName,
								MountPath: javaInstrMountPath,
							}},
						},
						{
							Name:    nodejsInitContainerName,
							Image:   "otel/nodejs:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nodejsVolumeName,
								MountPath: nodejsInstrMountPath,
							}},
						},
						{
							Name:    pythonInitContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", pythonInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      pythonVolumeName,
								MountPath: pythonInstrMountPath,
							}},
						},
						{
							Name:    dotnetInitContainerName,
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: dotnetInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "dotnet1",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=dotnet1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).dotnet1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
						{
							Name: "dotnet2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "dotnet2",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=dotnet2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).dotnet2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
						{
							Name: "java1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaAgent,
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "java1",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=java1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).java1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
							},
						},
						{
							Name: "java2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaAgent,
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "java2",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=java2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).java2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
							},
						},
						{
							Name: "nodejs1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "nodejs1",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=nodejs1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).nodejs1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
						{
							Name: "nodejs2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
								},
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "nodejs2",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=nodejs2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).nodejs2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
						{
							Name: "python1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "python1",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=python1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).python1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pythonVolumeName,
									MountPath: pythonInstrMountPath,
								},
							},
						},
						{
							Name: "python2",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "python2",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=python2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-multi-containers.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).python2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pythonVolumeName,
									MountPath: pythonInstrMountPath,
								},
							},
						},
					},
				},
			},
			config: config.New(
				config.WithEnableMultiInstrumentation(true),
				config.WithEnablePythonInstrumentation(true),
				config.WithEnableDotNetInstrumentation(true),
				config.WithEnableNodeJSInstrumentation(true),
			),
		},
		{
			name: "multi instrumentation for multiple containers feature gate disabled, multiple instrumentation annotations set",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-multi-containers-dis-cn",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-multi-containers-dis-cn",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Java: v1alpha1.Java{
						Image: "otel/java:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectJava:                 "true",
						annotationInjectNodeJS:               "true",
						annotationInjectPython:               "true",
						annotationInjectDotnetContainersName: "dotnet1,dotnet2",
						annotationInjectJavaContainersName:   "java1,java2",
						annotationInjectNodeJSContainersName: "nodejs1,nodejs2",
						annotationInjectPythonContainersName: "python1,python2",
						annotationInjectContainerName:        "should-not-be-instrumented1,should-not-be-instrumented2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "dotnet2",
						},
						{
							Name: "java1",
						},
						{
							Name: "java2",
						},
						{
							Name: "nodejs1",
						},
						{
							Name: "nodejs2",
						},
						{
							Name: "python1",
						},
						{
							Name: "python2",
						},
						{
							Name: "should-not-be-instrumented1",
						},
						{
							Name: "should-not-be-instrumented2",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectJava:                 "true",
						annotationInjectNodeJS:               "true",
						annotationInjectPython:               "true",
						annotationInjectDotnetContainersName: "dotnet1,dotnet2",
						annotationInjectJavaContainersName:   "java1,java2",
						annotationInjectNodeJSContainersName: "nodejs1,nodejs2",
						annotationInjectPythonContainersName: "python1,python2",
						annotationInjectContainerName:        "should-not-be-instrumented1,should-not-be-instrumented2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "dotnet2",
						},
						{
							Name: "java1",
						},
						{
							Name: "java2",
						},
						{
							Name: "nodejs1",
						},
						{
							Name: "nodejs2",
						},
						{
							Name: "python1",
						},
						{
							Name: "python2",
						},
						{
							Name: "should-not-be-instrumented1",
						},
						{
							Name: "should-not-be-instrumented2",
						},
					},
				},
			},
			config: config.New(config.WithEnableMultiInstrumentation(false), config.WithEnableJavaInstrumentation(false)),
		},
		{
			name: "multi instrumentation feature gate enabled, multiple instrumentation annotations set, no containers",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-multi-containers-no-cont",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-multi-containers-no-cont",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Java: v1alpha1.Java{
						Image: "otel/java:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
						annotationInjectJava:   "true",
						annotationInjectNodeJS: "true",
						annotationInjectPython: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "dotnet2",
						},
						{
							Name: "java1",
						},
						{
							Name: "java2",
						},
						{
							Name: "nodejs1",
						},
						{
							Name: "nodejs2",
						},
						{
							Name: "python1",
						},
						{
							Name: "python2",
						},
						{
							Name: "should-not-be-instrumented1",
						},
						{
							Name: "should-not-be-instrumented2",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
						annotationInjectJava:   "true",
						annotationInjectNodeJS: "true",
						annotationInjectPython: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "dotnet2",
						},
						{
							Name: "java1",
						},
						{
							Name: "java2",
						},
						{
							Name: "nodejs1",
						},
						{
							Name: "nodejs2",
						},
						{
							Name: "python1",
						},
						{
							Name: "python2",
						},
						{
							Name: "should-not-be-instrumented1",
						},
						{
							Name: "should-not-be-instrumented2",
						},
					},
				},
			},
			config: config.New(config.WithEnableMultiInstrumentation(true), config.WithEnableJavaInstrumentation(false)),
		},
		{
			name: "multi instrumentation feature gate enabled, single instrumentation annotation set, no containers",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-single-container-no-cont",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-single-container-no-cont",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Java: v1alpha1.Java{
						Image: "otel/java:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Python: v1alpha1.Python{
						Image: "otel/python:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
						},
						{
							Name: "should-not-be-instrumented1",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet: "true",
					},
				},
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
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      dotnetVolumeName,
								MountPath: dotnetInstrMountPath,
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
							Env: []corev1.EnvVar{
								{
									Name: "OTEL_NODE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "OTEL_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
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
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "dotnet1",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=dotnet1,k8s.namespace.name=multi-instrumentation-single-container-no-cont,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=multi-instrumentation-single-container-no-cont.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).dotnet1",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dotnetVolumeName,
									MountPath: dotnetInstrMountPath,
								},
							},
						},
						{
							Name: "should-not-be-instrumented1",
						},
					},
				},
			},
			config: config.New(
				config.WithEnableMultiInstrumentation(true),
				config.WithEnableDotNetInstrumentation(true),
			),
		},
		{
			name: "multi instrumentation feature gate disabled, instrumentation feature gate disabled and annotation set, multiple specific containers set",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-single-container-spec-cont",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-single-container-spec-cont",
				},
				Spec: v1alpha1.InstrumentationSpec{
					DotNet: v1alpha1.DotNet{
						Image: "otel/dotnet:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					NodeJS: v1alpha1.NodeJS{
						Image: "otel/nodejs:1",
						Env: []corev1.EnvVar{
							{
								Name:  "OTEL_LOG_LEVEL",
								Value: "debug",
							},
						},
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectDotnetContainersName: "dotnet1",
						annotationInjectNodeJSContainersName: "should-not-be-instrumented1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST",
									Value: "debug",
								},
							},
						},
						{
							Name: "should-not-be-instrumented1",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST",
									Value: "debug",
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectDotNet:               "true",
						annotationInjectDotnetContainersName: "dotnet1",
						annotationInjectNodeJSContainersName: "should-not-be-instrumented1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dotnet1",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST",
									Value: "debug",
								},
							},
						},
						{
							Name: "should-not-be-instrumented1",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST",
									Value: "debug",
								},
							},
						},
					},
				},
			},
			config: config.New(
				config.WithEnableMultiInstrumentation(true),
				config.WithEnableDotNetInstrumentation(false),
				config.WithEnableNodeJSInstrumentation(false),
			),
		},
		{
			name: "secret and configmap does not exists",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "error-missing-secrets",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "error-missing-secrets",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
						TLS: &v1alpha1.TLS{
							SecretName:    "my-certs",
							ConfigMapName: "my-ca-bundle",
							CA:            "ca.crt",
							Cert:          "cert.crt",
							Key:           "key.key",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
					Namespace: "error-missing-secrets",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			config: config.New(),
			err:    "secret error-missing-secrets/my-certs with certificates does not exists: secrets \"my-certs\" not found\nconfigmap error-missing-secrets/my-ca-bundle with CA certificate does not exists: configmaps \"my-ca-bundle\" not found",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mutator := NewMutator(logr.Discard(), k8sClient, record.NewFakeRecorder(100), test.config)
			require.NotNil(t, mutator)
			if test.setFeatureGates != nil {
				test.setFeatureGates(t)
			}

			err := k8sClient.Create(context.Background(), &test.ns)
			require.NoError(t, err)
			defer func() {
				_ = k8sClient.Delete(context.Background(), &test.ns)
			}()
			if test.secret != nil {
				err = k8sClient.Create(context.Background(), test.secret)
				require.NoError(t, err)
				defer func() {
					_ = k8sClient.Delete(context.Background(), test.secret)
				}()
			}
			if test.configMap != nil {
				err = k8sClient.Create(context.Background(), test.configMap)
				require.NoError(t, err)
				defer func() {
					_ = k8sClient.Delete(context.Background(), test.configMap)
				}()
			}

			err = k8sClient.Create(context.Background(), &test.inst)
			require.NoError(t, err)

			pod, err := mutator.Mutate(context.Background(), test.ns, test.pod)
			if test.err == "" {
				require.NoError(t, err)
				assert.Equal(t, test.expected, pod)
			} else {
				assert.Contains(t, err.Error(), test.err)
			}
		})
	}
}

func TestContainerNamesConfiguredForMultipleInstrumentations(t *testing.T) {
	tests := []struct {
		name             string
		instrumentations languageInstrumentations
		expectedStatus   bool
		expectedMsg      error
	}{
		{
			name: "Single instrumentation enabled without containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: true,
			expectedMsg:    nil,
		},
		{
			name: "Multiple instrumentations enabled with containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"java"}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"nodejs"}},
			},
			expectedStatus: true,
			expectedMsg:    nil,
		},
		{
			name: "Multiple instrumentations enabled without containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("incorrect instrumentation configuration - please provide container names for all instrumentations"),
		},
		{
			name: "Multiple instrumentations enabled with containers for single instrumentation",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"test"}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("incorrect instrumentation configuration - please provide container names for all instrumentations"),
		},
		{
			name: "Disabled instrumentations",
			instrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("instrumentation configuration not provided"),
		},
		{
			name: "Multiple instrumentations enabled with duplicated containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"app", "app1", "java"}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"app1", "app", "nodejs"}},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("duplicated container names detected: [app app1]"),
		},
		{
			name: "Multiple instrumentations enabled with duplicated containers for single instrumentation",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"app", "app", "java"}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"nodejs"}},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("duplicated container names detected: [app]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, msg := test.instrumentations.areInstrumentedContainersCorrect()
			assert.Equal(t, test.expectedStatus, ok)
			assert.Equal(t, test.expectedMsg, msg)
		})
	}
}

func TestInstrumentationLanguageContainersSet(t *testing.T) {
	tests := []struct {
		name                     string
		instrumentations         languageInstrumentations
		containers               string
		pod                      corev1.Pod
		ns                       corev1.Namespace
		expectedInstrumentations languageInstrumentations
	}{
		{
			name: "Set containers for enabled instrumentation",
			instrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectContainerName: "python,python1",
					},
				},
			},
			ns: corev1.Namespace{},
			expectedInstrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: []string{"python", "python1"}},
			},
		},
		{
			name:             "Set containers when all instrumentations disabled",
			instrumentations: languageInstrumentations{},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectContainerName: "python,python1",
					},
				},
			},
			expectedInstrumentations: languageInstrumentations{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.instrumentations.setCommonInstrumentedContainers(test.ns, test.pod)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedInstrumentations, test.instrumentations)
		})
	}
}
