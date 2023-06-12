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
									Name:  envDotNetCoreClrEnableProfiling,
									Value: dotNetCoreClrEnableProfilingEnabled,
								},
								{
									Name:  envDotNetCoreClrProfiler,
									Value: dotNetCoreClrProfilerID,
								},
								{
									Name:  envDotNetCoreClrProfilerPath,
									Value: dotNetCoreClrProfilerPath,
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
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_PTRACE"},
								},
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
				require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), true))
				t.Cleanup(func() {
					require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.EnableGoAutoInstrumentationSupport.ID(), originalVal))
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
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "otel-apache-agent",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
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
								"cp -ar /opt/opentelemetry/* /opt/opentelemetry-webserver/agent && export agentLogDir=$(echo \"/opt/opentelemetry-webserver/agent/logs\" | sed 's,/,\\\\/,g') && cat /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml &&echo \"$OTEL_APACHE_AGENT_CONF\" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && sed -i 's/<<SID-PLACEHOLDER>>/'${APACHE_SERVICE_INSTANCE_ID}'/g' /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && echo 'Include /usr/local/apache2/conf/opentemetry_agent.conf' >> /opt/opentelemetry-webserver/source-conf/httpd.conf"},
							Env: []corev1.EnvVar{
								{
									Name:  apacheAttributesEnvVar,
									Value: "\n#Load the Otel Webserver SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_common.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_resources.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_trace.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_otlp_recordable.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_ostream_span.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_otlp_grpc.so\n#Load the Otel ApacheModule SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_webserver_sdk.so\n#Load the Apache Module. In this example for Apache 2.4\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Load the Apache Module. In this example for Apache 2.2\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel22.so\nLoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Attributes\nApacheModuleEnabled ON\nApacheModuleOtelExporterEndpoint http://collector:12345\nApacheModuleOtelSpanExporter otlp\nApacheModuleResolveBackends  ON\nApacheModuleServiceInstanceId <<SID-PLACEHOLDER>>\nApacheModuleServiceName app\nApacheModuleServiceNamespace \nApacheModuleTraceAsError  ON\n",
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
									MountPath: apacheConfigDirectory,
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
