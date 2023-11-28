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
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestMutatePod(t *testing.T) {
	mutator := NewMutator(logr.Discard(), k8sClient, record.NewFakeRecorder(100))
	require.NotNil(t, mutator)

	true := true
	zero := int64(0)

	tests := []struct {
		name            string
		err             string
		pod             corev1.Pod
		expected        corev1.Pod
		inst            v1alpha1.Instrumentation
		ns              corev1.Namespace
		setFeatureGates func(t *testing.T)
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
									Name:      javaVolumeName,
									MountPath: javaInstrMountPath,
								},
							},
						},
					},
				},
			},
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
									Value: "k8s.container.name=app1,k8s.namespace.name=javaagent-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=app2,k8s.namespace.name=javaagent-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableJavaAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableJavaAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableJavaAutoInstrumentationSupport.ID(), originalVal))
				})
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", nodejsInstrMountPath},
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
									Name:      nodejsVolumeName,
									MountPath: nodejsInstrMountPath,
								},
							},
						},
					},
				},
			},
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", nodejsInstrMountPath},
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
									Value: "k8s.container.name=app1,k8s.namespace.name=nodejs-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=app2,k8s.namespace.name=nodejs-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableNodeJSAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNodeJSAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNodeJSAutoInstrumentationSupport.ID(), originalVal))
				})
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", pythonInstrMountPath},
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
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
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
									Value: "k8s.container.name=app,k8s.namespace.name=python,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", pythonInstrMountPath},
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
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
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
									Value: "k8s.container.name=app1,k8s.namespace.name=python-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://localhost:4318",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
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
									Value: "k8s.container.name=app2,k8s.namespace.name=python-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnablePythonAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnablePythonAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnablePythonAutoInstrumentationSupport.ID(), originalVal))
				})
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=app,k8s.namespace.name=dotnet,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=app,k8s.namespace.name=dotnet-by-namespace-annotation,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=app1,k8s.namespace.name=dotnet-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=app2,k8s.namespace.name=dotnet-multiple-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableDotnetAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableDotnetAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableDotnetAutoInstrumentationSupport.ID(), originalVal))
				})
			},
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
									Value: "k8s.container.name=app,k8s.namespace.name=go,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableGoAutoInstrumentationSupport.IsEnabled()
				mtVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()

				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))

				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), originalVal))
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), mtVal))

				})
			},
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
								"cp -r /opt/opentelemetry/* /opt/opentelemetry-webserver/agent && export agentLogDir=$(echo \"/opt/opentelemetry-webserver/agent/logs\" | sed 's,/,\\\\/,g') && cat /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml &&echo \"$OTEL_APACHE_AGENT_CONF\" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && sed -i 's/<<SID-PLACEHOLDER>>/'${APACHE_SERVICE_INSTANCE_ID}'/g' /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && echo 'Include /usr/local/apache2/conf/opentemetry_agent.conf' >> /opt/opentelemetry-webserver/source-conf/httpd.conf"},
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
									Value: "k8s.container.name=app,k8s.namespace.name=apache-httpd,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
						},
					},
				},
			},
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableApacheHTTPAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), originalVal))
				})
			},
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableApacheHTTPAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableApacheHTTPAutoInstrumentationSupport.ID(), originalVal))
				})
			},
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
									Name: "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=nginx,k8s.namespace.name=req-namespace,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=my-nginx-6c44bcbdd,service.instance.id=req-namespace.my-nginx-6c44bcbdd.nginx",
								},
							},
						},
					},
				},
			},
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableNginxAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), originalVal))
				})
			},
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableNginxAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableNginxAutoInstrumentationSupport.ID(), originalVal))
				})
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", nodejsInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nodejsVolumeName,
								MountPath: nodejsInstrMountPath,
							}},
						},
						{
							Name:    pythonInitContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", pythonInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      pythonVolumeName,
								MountPath: pythonInstrMountPath,
							}},
						},
						{
							Name:    dotnetInitContainerName,
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=dotnet1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=dotnet2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
									Value: "k8s.container.name=java1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
									Value: "k8s.container.name=java2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=nodejs1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=nodejs2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
									Value: "http/protobuf",
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
									Value: "k8s.container.name=python1,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
									Value: "http/protobuf",
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
									Value: "k8s.container.name=python2,k8s.namespace.name=multi-instrumentation-multi-containers,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
		},
		{
			name: "multi instrumentation for multiple containers feature gate enabled, container-names not used",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-instrumentation-multi-containers-cn",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "multi-instrumentation-multi-containers-cn",
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", nodejsInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      nodejsVolumeName,
								MountPath: nodejsInstrMountPath,
							}},
						},
						{
							Name:    pythonInitContainerName,
							Image:   "otel/python:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", pythonInstrMountPath},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      pythonVolumeName,
								MountPath: pythonInstrMountPath,
							}},
						},
						{
							Name:    dotnetInitContainerName,
							Image:   "otel/dotnet:1",
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=dotnet1,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=dotnet2,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
									Value: "k8s.container.name=java1,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
									Value: "k8s.container.name=java2,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=nodejs1,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=nodejs2,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
									Value: "http/protobuf",
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
									Value: "k8s.container.name=python1,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Name:  "OTEL_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "PYTHONPATH",
									Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
									Value: "http/protobuf",
								},
								{
									Name:  "OTEL_METRICS_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
									Value: "http/protobuf",
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
									Value: "k8s.container.name=python2,k8s.namespace.name=multi-instrumentation-multi-containers-cn,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
							Name: "should-not-be-instrumented1",
						},
						{
							Name: "should-not-be-instrumented2",
						},
					},
				},
			},
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
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
							Command: []string{"cp", "-a", "/autoinstrumentation/.", dotnetInstrMountPath},
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
									Value: "k8s.container.name=dotnet1,k8s.namespace.name=multi-instrumentation-single-container-no-cont,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
			setFeatureGates: func(t *testing.T) {
				originalVal := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalVal))
				})
			},
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
			setFeatureGates: func(t *testing.T) {
				originalValMultiInstr := featuregate.EnableMultiInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), true))
				originalValDotNetInstr := featuregate.EnableDotnetAutoInstrumentationSupport.IsEnabled()
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableDotnetAutoInstrumentationSupport.ID(), false))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableMultiInstrumentationSupport.ID(), originalValMultiInstr))
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableDotnetAutoInstrumentationSupport.ID(), originalValDotNetInstr))
				})
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.setFeatureGates != nil {
				test.setFeatureGates(t)
			}

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

func TestSingleInstrumentationEnabled(t *testing.T) {
	tests := []struct {
		name             string
		instrumentations languageInstrumentations
		expectedStatus   bool
		expectedMsg      string
	}{
		{
			name: "Single instrumentation enabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: true,
			expectedMsg:    "Java",
		},
		{
			name: "Multiple instrumentations enabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    "",
		},
		{
			name: "Instrumentations disabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: nil},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: false,
			expectedMsg:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok := test.instrumentations.isSingleInstrumentationEnabled()
			assert.Equal(t, test.expectedStatus, ok)
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
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "nodejs"},
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
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "test"},
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
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app,app1,java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app1,app,nodejs"},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("duplicated container names detected: [app app1]"),
		},
		{
			name: "Multiple instrumentations enabled with duplicated containers for single instrumentation",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app,app,java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "nodejs"},
			},
			expectedStatus: false,
			expectedMsg:    fmt.Errorf("duplicated container names detected: [app]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, msg := test.instrumentations.areContainerNamesConfiguredForMultipleInstrumentations()
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
		expectedInstrumentations languageInstrumentations
	}{
		{
			name: "Set containers for enabled instrumentation",
			instrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			containers: "python,python1",
			expectedInstrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "python,python1"},
			},
		},
		{
			name:                     "Set containers when all instrumentations disabled",
			instrumentations:         languageInstrumentations{},
			containers:               "python,python1",
			expectedInstrumentations: languageInstrumentations{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.instrumentations.setInstrumentationLanguageContainers(test.containers)
			assert.Equal(t, test.expectedInstrumentations, test.instrumentations)
		})
	}
}
