// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestMutatePod(t *testing.T) {
	mutator := NewMutator(logr.Discard(), k8sClient)
	require.NotNil(t, mutator)

	tests := []struct {
		name     string
		ns       corev1.Namespace
		pod      corev1.Pod
		inst     v1alpha1.Instrumentation
		expected corev1.Pod
		err      string
	}{
		{
			name: "javaagent injection, true",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "javaagent",
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
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation/javaagent.jar"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
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
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
									Value: "k8s.container.name=app,k8s.namespace.name=javaagent,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation",
									MountPath: "/otel-auto-instrumentation",
								},
							},
						},
					},
				},
			},
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
							Name: "opentelemetry-auto-instrumentation",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "otel/nodejs:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
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
									Value: "k8s.container.name=app,k8s.namespace.name=nodejs,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation",
									MountPath: "/otel-auto-instrumentation",
								},
							},
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
								Value: "otlp_proto_http",
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
							Name: "opentelemetry-auto-instrumentation",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp_proto_http",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
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
									Value: "k8s.container.name=app,k8s.namespace.name=python,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation",
									MountPath: "/otel-auto-instrumentation",
								},
							},
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
					Volumes: []corev1.Volume{
						{
							Name: "opentelemetry-auto-instrumentation",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    initContainerName,
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      volumeName,
								MountPath: "/otel-auto-instrumentation",
							}},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4317",
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
									Value: "k8s.container.name=app,k8s.namespace.name=dotnet,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opentelemetry-auto-instrumentation",
									MountPath: "/otel-auto-instrumentation",
								},
							},
						},
					},
				},
			},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := k8sClient.Create(context.Background(), &test.ns)
			require.NoError(t, err)
			defer func() {
				_ = k8sClient.Delete(context.Background(), &test.ns)
			}()
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
