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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var testResourceRequirements = corev1.ResourceRequirements{
	Limits: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	},
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	},
}

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
							Name:  "application-name",
							Image: "app:latest",
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
							Name:  "application-name",
							Image: "app:latest",
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
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.deployment.uid=depuid,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app,k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,k8s.replicaset.uid=rsuid,service.version=latest",
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
							Image: "app:latest",
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
									Value: "foo=bar,k8s.container.name=other,service.version=explicitly_set,",
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
							Image: "app:latest",
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
									Value: "foo=bar,k8s.container.name=other,service.version=explicitly_set,fromcr=val,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app",
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
							Name:  "application-name",
							Image: "app:latest",
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
							Name:  "application-name",
							Image: "app:latest",
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
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=app,k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,service.version=latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "SDK image with port number, no version",
			inst: v1alpha1.Instrumentation{},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "fictional.registry.example:10443/imagename",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "fictional.registry.example:10443/imagename",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "",
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
									Value: "k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "SDK image with port number, with version",
			inst: v1alpha1.Instrumentation{},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "fictional.registry.example:10443/imagename:latest",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "fictional.registry.example:10443/imagename:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "",
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
									Value: "k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
			pod := inj.injectCommonSDKConfig(context.Background(), test.inst, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: test.pod.Namespace}}, test.pod, 0, 0)
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
				Image:     "img:1",
				Resources: testResourceRequirements,
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
						Name:  "app",
						Image: "app:latest",
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
					Resources: testResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
				Image:     "img:1",
				Resources: testResourceRequirements,
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
						Name:  "app",
						Image: "app:latest",
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
					Resources: testResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
						Name:  "app",
						Image: "app:latest",
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
					Name:  "app",
					Image: "app:latest",
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
						Name:  "app",
						Image: "app:latest",
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
					Name:  "app",
					Image: "app:latest",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/otel-auto-instrumentation",
						},
					},
					Env: []corev1.EnvVar{
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
						},
					},
				},
			},
		},
	}, pod)
}

func TestInjectGo(t *testing.T) {
	falsee := false
	true := true
	zero := int64(0)

	tests := []struct {
		name     string
		insts    languageInstrumentations
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name: "shared process namespace disabled",
			insts: languageInstrumentations{
				Go: &v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						Go: v1alpha1.Go{
							Image: "otel/go:1",
						},
					},
				},
			},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &falsee,
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &falsee,
					Containers: []corev1.Container{
						{
							Name: "app",
						},
					},
				},
			},
		},
		{
			name: "OTEL_GO_AUTO_TARGET_EXE not set",
			insts: languageInstrumentations{
				Go: &v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						Go: v1alpha1.Go{
							Image: "otel/go:1",
						},
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
			name: "OTEL_GO_AUTO_TARGET_EXE set by inst",
			insts: languageInstrumentations{
				Go: &v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						Go: v1alpha1.Go{
							Image: "otel/go:1",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_GO_AUTO_TARGET_EXE",
									Value: "foo",
								},
							},
						},
					},
				},
			},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			},
			expected: corev1.Pod{
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
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
									Value: "foo",
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
									Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
		},
		{
			name: "OTEL_GO_AUTO_TARGET_EXE set by annotation",
			insts: languageInstrumentations{
				Go: &v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						Go: v1alpha1.Go{
							Image: "otel/go:1",
						},
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/otel-go-auto-target-exe": "foo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			},
			expected: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/otel-go-auto-target-exe": "foo",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &true,
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
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
									Value: "foo",
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
									Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inj := sdkInjector{
				logger: logr.Discard(),
			}
			pod := inj.inject(context.Background(), test.insts, corev1.Namespace{}, test.pod, "")
			assert.Equal(t, test.expected, pod)
		})
	}
}

func TestInjectApacheHttpd(t *testing.T) {

	tests := []struct {
		name     string
		insts    languageInstrumentations
		pod      corev1.Pod
		expected corev1.Pod
	}{
		{
			name: "injection enabled, exporter set",
			insts: languageInstrumentations{
				ApacheHttpd: &v1alpha1.Instrumentation{
					Spec: v1alpha1.InstrumentationSpec{
						ApacheHttpd: v1alpha1.ApacheHttpd{
							Image: "img:1",
						},
						Exporter: v1alpha1.Exporter{
							Endpoint: "https://collector:4318",
						},
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
							Image:   "img:1",
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								"cp -ar /opt/opentelemetry/* /opt/opentelemetry-webserver/agent && export agentLogDir=$(echo \"/opt/opentelemetry-webserver/agent/logs\" | sed 's,/,\\\\/,g') && cat /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > /opt/opentelemetry-webserver/agent/conf/appdynamics_sdk_log4cxx.xml &&echo \"$OTEL_APACHE_AGENT_CONF\" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && sed -i 's/<<SID-PLACEHOLDER>>/'${APACHE_SERVICE_INSTANCE_ID}'/g' /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && echo 'Include /usr/local/apache2/conf/opentemetry_agent.conf' >> /opt/opentelemetry-webserver/source-conf/httpd.conf"},
							Env: []corev1.EnvVar{
								{
									Name:  apacheAttributesEnvVar,
									Value: "\n#Load the Otel Webserver SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_common.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_resources.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_trace.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_otlp_recordable.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_ostream_span.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_otlp_grpc.so\n#Load the Otel ApacheModule SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_webserver_sdk.so\n#Load the Apache Module. In this example for Apache 2.4\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Load the Apache Module. In this example for Apache 2.2\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel22.so\nLoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Attributes\nApacheModuleEnabled ON\nApacheModuleOtelExporterEndpoint https://collector:4318\nApacheModuleOtelSpanExporter otlp\nApacheModuleResolveBackends  ON\nApacheModuleServiceInstanceId <<SID-PLACEHOLDER>>\nApacheModuleServiceName app\nApacheModuleServiceNamespace \nApacheModuleTraceAsError  ON\n",
								},
								{Name: apacheServiceInstanceIdEnvVar,
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
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inj := sdkInjector{
				logger: logr.Discard(),
			}
			pod := inj.inject(context.Background(), test.insts, corev1.Namespace{}, test.pod, "")
			assert.Equal(t, test.expected, pod)
		})
	}
}

func TestInjectSdkOnly(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "https://collector:4318",
			},
		},
	}
	insts := languageInstrumentations{
		Sdk: &inst,
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
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, "")
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
					Env: []corev1.EnvVar{
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
							Value: "k8s.container.name=app,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.version=latest",
						},
					},
				},
			},
		},
	}, pod)
}
