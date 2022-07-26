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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestSDKInjection(t *testing.T) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "project1",
		},
	}
	err := k8sClient.Create(context.Background(), &ns)
	require.NoError(t, err)
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "project1",
			Name:      "my-deployment",
			UID:       "depuid",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "my"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "my"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "foo:bar"}},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), &dep)
	require.NoError(t, err)
	rs := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-replicaset",
			Namespace: "project1",
			UID:       "rsuid",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
					Name:       "my-deployment",
					UID:        "depuid",
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "my"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "my"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "foo:bar"}},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), &rs)
	require.NoError(t, err)

	tests := []struct {
		name     string
		inst     v1alpha1.Instrumentation
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name: "SDK env vars not defined",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Exporter: v1alpha1.Exporter{
						Endpoint: "https://collector:4317",
					},
					Resource: v1alpha1.Resource{
						AddK8sUIDAttributes: true,
					},
					Propagators: []v1alpha1.Propagator{"b3", "jaeger"},
					Sampler: v1alpha1.Sampler{
						Type:     "parentbased_traceidratio",
						Argument: "0.25",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
					UID:       "pod-uid",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ReplicaSet",
							Name:       "my-replicaset",
							UID:        "rsuid",
							APIVersion: "apps/v1",
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "application-name",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
					UID:       "pod-uid",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ReplicaSet",
							Name:       "my-replicaset",
							UID:        "rsuid",
							APIVersion: "apps/v1",
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "application-name",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "my-deployment",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "https://collector:4317",
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
									Name:  "OTEL_PROPAGATORS",
									Value: "b3,jaeger",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "0.25",
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.deployment.uid=depuid,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app,k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,k8s.replicaset.uid=rsuid",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "SDK env vars defined",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Exporter: v1alpha1.Exporter{
						Endpoint: "https://collector:4317",
					},
					Resource: v1alpha1.Resource{
						Attributes: map[string]string{
							"fromcr": "val",
						},
					},
					Propagators: []v1alpha1.Propagator{"jaeger"},
					Sampler: v1alpha1.Sampler{
						Type:     "parentbased_traceidratio",
						Argument: "0.25",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "explicitly_set",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "explicitly_set",
								},
								{
									Name:  "OTEL_PROPAGATORS",
									Value: "b3",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "always_on",
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "foo=bar,k8s.container.name=other,",
								},
							},
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "explicitly_set",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "explicitly_set",
								},
								{
									Name:  "OTEL_PROPAGATORS",
									Value: "b3",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "always_on",
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
									Value: "foo=bar,k8s.container.name=other,fromcr=val,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Empty instrumentation spec",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
					UID:       "pod-uid",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ReplicaSet",
							Name:       "my-replicaset",
							UID:        "rsuid",
							APIVersion: "apps/v1",
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "application-name",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
					UID:       "pod-uid",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ReplicaSet",
							Name:       "my-replicaset",
							UID:        "rsuid",
							APIVersion: "apps/v1",
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "application-name",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "my-deployment",
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
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app,k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset",
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
			inj := sdkInjector{
				client: k8sClient,
			}
			pod := inj.injectCommonSDKConfig(context.Background(), test.inst, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: test.pod.Namespace}}, test.pod, 0)
			_, err = json.MarshalIndent(pod, "", "  ")
			assert.NoError(t, err)
			assert.Equal(t, test.expected, pod)
		})
	}
}

func TestInjectJava(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		Spec: v1alpha1.InstrumentationSpec{
			Java: v1alpha1.Java{
				Image: "img:1",
			},
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4317",
			},
		},
	}
	insts := languageInstrumentations{
		Java: &inst,
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	pod := inj.inject(context.Background(), insts,
		corev1.Namespace{},
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app",
					},
				},
			},
		}, "")
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   "img:1",
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/otel-auto-instrumentation",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "JAVA_TOOL_OPTIONS",
							Value: javaJVMArgument,
						},
						{
							Name:  "OTEL_SERVICE_NAME",
							Value: "app",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "https://collector:4317",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
						},
					},
				},
			},
		},
	}, pod)
}

func TestInjectNodeJS(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		Spec: v1alpha1.InstrumentationSpec{
			NodeJS: v1alpha1.NodeJS{
				Image: "img:1",
			},
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
			},
		},
	}
	insts := languageInstrumentations{
		NodeJS: &inst,
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	pod := inj.inject(context.Background(), insts,
		corev1.Namespace{},
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app",
					},
				},
			},
		}, "")
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   "img:1",
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/otel-auto-instrumentation",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "NODE_OPTIONS",
							Value: nodeRequireArgument,
						},
						{
							Name:  "OTEL_SERVICE_NAME",
							Value: "app",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "https://collector:4318",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
						},
					},
				},
			},
		},
	}, pod)
}

func TestInjectPython(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		Spec: v1alpha1.InstrumentationSpec{
			Python: v1alpha1.Python{
				Image: "img:1",
			},
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
			},
		},
	}
	insts := languageInstrumentations{
		Python: &inst,
	}

	inj := sdkInjector{
		logger: logr.Discard(),
	}
	pod := inj.inject(context.Background(), insts,
		corev1.Namespace{},
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app",
					},
				},
			},
		}, "")
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   "img:1",
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/otel-auto-instrumentation",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "PYTHONPATH",
							Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
						},
						{
							Name:  "OTEL_TRACES_EXPORTER",
							Value: "otlp_proto_http",
						},
						{
							Name:  "OTEL_SERVICE_NAME",
							Value: "app",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "https://collector:4318",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
						},
					},
				},
			},
		},
	}, pod)
}

func TestInjectDotNet(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		Spec: v1alpha1.InstrumentationSpec{
			DotNet: v1alpha1.DotNet{
				Image: "img:1",
			},
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
			},
		},
	}
	insts := languageInstrumentations{
		DotNet: &inst,
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	pod := inj.inject(context.Background(), insts,
		corev1.Namespace{},
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app",
					},
				},
			},
		}, "")
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   "img:1",
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/otel-auto-instrumentation",
						},
					},
					Env: []corev1.EnvVar{
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
							Name:  "OTEL_SERVICE_NAME",
							Value: "app",
						},
						{
							Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
							Value: "https://collector:4318",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
						},
					},
				},
			},
		},
	}, pod)
}
