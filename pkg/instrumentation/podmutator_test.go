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
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/api/instrumentation/v1alpha1"
)

var k8sClient client.Client
var testEnv *envtest.Environment
var testScheme = scheme.Scheme

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start testEnv: %v", err)
		os.Exit(1)
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		fmt.Printf("failed to setup a Kubernetes client: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	err = testEnv.Stop()
	if err != nil {
		fmt.Printf("failed to stop testEnv: %v", err)
		os.Exit(1)
	}

	os.Exit(code)
}

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
					Java: v1alpha1.JavaSpec{
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
						annotationInject:   "true",
						annotationLanguage: "java",
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
						annotationInject:   "true",
						annotationLanguage: "java",
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
							Image:   "otel/java:1",
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
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name:  "JAVA_TOOL_OPTIONS",
									Value: javaJVMArgument,
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
					Java: v1alpha1.JavaSpec{
						Image: "otel/nodejs:1",
					},
					Exporter: v1alpha1.Exporter{
						Endpoint: "http://collector:12345",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInject:   "true",
						annotationLanguage: "nodejs",
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
						annotationInject:   "true",
						annotationLanguage: "nodejs",
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
							Image:   "otel/java:1",
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
									Name:  "OTEL_SERVICE_NAME",
									Value: "app",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:12345",
								},
								{
									Name:  "NODE_OPTIONS",
									Value: nodeRequireArgument,
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
			name: "missing language",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-language",
				},
			},
			inst: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-inst",
					Namespace: "missing-language",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.JavaSpec{
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
						annotationInject: "true",
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
					Java: v1alpha1.JavaSpec{
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
					Java: v1alpha1.JavaSpec{
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
						annotationInject: "false",
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
						annotationInject: "false",
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
					Java: v1alpha1.JavaSpec{
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
						annotationInject: "doesnotexists",
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
