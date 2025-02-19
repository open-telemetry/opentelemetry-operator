// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var testNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ns",
	},
}

var defaultVolumeLimitSize = resource.MustParse("200Mi")

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
						Attributes: map[string]string{
							"foo": "hidden",
						},
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
					Labels: map[string]string{
						"app.kubernetes.io/name":    "app-name",
						"app.kubernetes.io/version": "v1",
						"app.kubernetes.io/part-of": "shop",
					},
					Annotations: map[string]string{
						"resource.opentelemetry.io/foo": "bar",
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
					Labels: map[string]string{
						"app.kubernetes.io/name":    "app-name",
						"app.kubernetes.io/version": "v1",
						"app.kubernetes.io/part-of": "shop",
					},
					Annotations: map[string]string{
						"resource.opentelemetry.io/foo": "bar",
					},
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
									Value: "foo=bar,k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.deployment.uid=depuid,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,k8s.replicaset.uid=rsuid,service.instance.id=project1.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).application-name,service.version=latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Resource attribute from CRD",
			inst: v1alpha1.Instrumentation{
				Spec: v1alpha1.InstrumentationSpec{
					Exporter: v1alpha1.Exporter{
						Endpoint: "https://collector:4317",
					},
					Resource: v1alpha1.Resource{
						AddK8sUIDAttributes: true,
						Attributes: map[string]string{
							"k8s.container.name":  "explicit-container",
							"k8s.deployment.name": "explicit-deployment",
							"k8s.deployment.uid":  "explicit-deployment-uid",
							"k8s.namespace.name":  "explicit-ns",
							"k8s.node.name":       "explicit-node",
							"k8s.pod.name":        "explicit-pod",
							"k8s.pod.uid":         "explicit-pod-uid",
							"k8s.replicaset.name": "explicit-replicaset",
							"k8s.replicaset.uid":  "explicit-replicaset-uid",
							"service.instance.id": "explicit-id",
							"service.version":     "explicit-version",
						},
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
					NodeName: "node-name",
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
					NodeName: "node-name",
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
									Name: "OTEL_RESOURCE_ATTRIBUTES_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
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
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.deployment.uid=depuid,k8s.namespace.name=project1,k8s.node.name=node-name,k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,k8s.replicaset.uid=rsuid,service.instance.id=project1.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).application-name,service.version=latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "SDK env vars not defined - use labels for resource attributes",
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
					Defaults: v1alpha1.Defaults{
						UseLabelsForResourceAttributes: true,
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
					Labels: map[string]string{
						"app.kubernetes.io/name":    "app-name",
						"app.kubernetes.io/version": "v1",
						"app.kubernetes.io/part-of": "shop",
					},
					Annotations: map[string]string{
						"resource.opentelemetry.io/foo": "bar",
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
					Labels: map[string]string{
						"app.kubernetes.io/name":    "app-name",
						"app.kubernetes.io/version": "v1",
						"app.kubernetes.io/part-of": "shop",
					},
					Annotations: map[string]string{
						"resource.opentelemetry.io/foo": "bar",
					},
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
									Value: "app-name",
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
									Value: "foo=bar,k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.deployment.uid=depuid,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,k8s.replicaset.uid=rsuid,service.instance.id=project1.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).application-name,service.namespace=shop,service.version=v1",
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
					Defaults: v1alpha1.Defaults{
						UseLabelsForResourceAttributes: true,
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
					Name:      "app",
					Labels: map[string]string{
						"app.kubernetes.io/name":    "not-used",
						"app.kubernetes.io/version": "not-used",
						"app.kubernetes.io/part-of": "not-used",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "app:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "explicit-name",
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
									Value: "foo=bar,k8s.container.name=other,service.version=explicit-version,service.namespace=explicit-ns,service.instance.id=explicit-id,",
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
					Labels: map[string]string{
						"app.kubernetes.io/name":    "not-used",
						"app.kubernetes.io/version": "not-used",
						"app.kubernetes.io/part-of": "not-used",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "app:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "OTEL_SERVICE_NAME",
									Value: "explicit-name",
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
									Value: "foo=bar,k8s.container.name=other,service.version=explicit-version,service.namespace=explicit-ns,service.instance.id=explicit-id,fromcr=val,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
									Value: "k8s.container.name=application-name,k8s.deployment.name=my-deployment,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),k8s.pod.uid=pod-uid,k8s.replicaset.name=my-replicaset,service.instance.id=project1.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).application-name,service.version=latest",
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
		{
			name: "Resource attribute propagate",
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
					Annotations: map[string]string{
						"resource.opentelemetry.io/fromtest": "val",
						"resource.opentelemetry.io/foo":      "test",
					},
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
					Annotations: map[string]string{
						"resource.opentelemetry.io/fromtest": "val",
						"resource.opentelemetry.io/foo":      "test",
					},
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
									Value: "foo=bar,k8s.container.name=other,service.version=explicitly_set,fromcr=val,fromtest=val,k8s.namespace.name=project1,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME)",
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
		Java: instrumentationWithContainers{
			Instrumentation: &inst,
			Containers:      []string{"app"},
		},
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	config := config.New()
	pod := inj.inject(context.Background(), insts,
		testNamespace,
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, config)
	assert.Equal(t, corev1.Pod{
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
					Image:   "img:1",
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
					Name:  "app",
					Image: "app:latest",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      javaVolumeName,
							MountPath: javaInstrMountPath,
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
							Name:  "JAVA_TOOL_OPTIONS",
							Value: javaAgent,
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
							Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
		NodeJS: instrumentationWithContainers{
			Instrumentation: &inst,
			Containers:      []string{"app"},
		},
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	config := config.New()
	pod := inj.inject(context.Background(), insts,
		testNamespace,
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, config)
	assert.Equal(t, corev1.Pod{
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
					Image:   "img:1",
					Command: []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      nodejsVolumeName,
						MountPath: nodejsInstrMountPath,
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
							Name:      nodejsVolumeName,
							MountPath: nodejsInstrMountPath,
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
							Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
		Python: instrumentationWithContainers{
			Instrumentation: &inst,
			Containers:      []string{"app"},
		},
	}

	inj := sdkInjector{
		logger: logr.Discard(),
	}
	config := config.New()
	pod := inj.inject(context.Background(), insts,
		testNamespace,
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, config)
	assert.Equal(t, corev1.Pod{
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
					Image:   "img:1",
					Command: []string{"cp", "-r", "/autoinstrumentation/.", pythonInstrMountPath},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      pythonVolumeName,
						MountPath: pythonInstrMountPath,
					}},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      pythonVolumeName,
							MountPath: pythonInstrMountPath,
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
							Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
		DotNet: instrumentationWithContainers{
			Instrumentation: &inst,
			Containers:      []string{"app"},
		},
	}
	inj := sdkInjector{
		logger: logr.Discard(),
	}
	config := config.New()
	pod := inj.inject(context.Background(), insts,
		testNamespace,
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, config)
	assert.Equal(t, corev1.Pod{
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
					Image:   "img:1",
					Command: []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      dotnetVolumeName,
						MountPath: dotnetInstrMountPath,
					}},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      dotnetVolumeName,
							MountPath: dotnetInstrMountPath,
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
							Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
		config   config.Config
	}{
		{
			name: "shared process namespace disabled",
			insts: languageInstrumentations{
				Go: instrumentationWithContainers{
					Containers: []string{"app"},
					Instrumentation: &v1alpha1.Instrumentation{
						Spec: v1alpha1.InstrumentationSpec{
							Go: v1alpha1.Go{
								Image: "otel/go:1",
							},
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
				Go: instrumentationWithContainers{
					Containers: []string{"app"},
					Instrumentation: &v1alpha1.Instrumentation{
						Spec: v1alpha1.InstrumentationSpec{
							Go: v1alpha1.Go{
								Image: "otel/go:1",
							},
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
				Go: instrumentationWithContainers{
					Containers: []string{"app"},
					Instrumentation: &v1alpha1.Instrumentation{
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
									Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
				Go: instrumentationWithContainers{
					Containers: []string{"app"},
					Instrumentation: &v1alpha1.Instrumentation{
						Spec: v1alpha1.InstrumentationSpec{
							Go: v1alpha1.Go{
								Image: "otel/go:1",
							},
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
									Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
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
			pod := inj.inject(context.Background(), test.insts, testNamespace, test.pod, test.config)
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
		config   config.Config
	}{
		{
			name: "injection enabled, exporter set",
			insts: languageInstrumentations{
				ApacheHttpd: instrumentationWithContainers{
					Instrumentation: &v1alpha1.Instrumentation{
						Spec: v1alpha1.InstrumentationSpec{
							ApacheHttpd: v1alpha1.ApacheHttpd{
								Image: "img:1",
							},
							Exporter: v1alpha1.Exporter{
								Endpoint: "https://collector:4318",
							},
						},
					},
					Containers: []string{"app"},
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
							Image:   "img:1",
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								"cp -r /opt/opentelemetry/* /opt/opentelemetry-webserver/agent && export agentLogDir=$(echo \"/opt/opentelemetry-webserver/agent/logs\" | sed 's,/,\\\\/,g') && cat /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml.template | sed 's/__agent_log_dir__/'${agentLogDir}'/g'  > /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml &&echo \"$OTEL_APACHE_AGENT_CONF\" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && sed -i 's/<<SID-PLACEHOLDER>>/'${APACHE_SERVICE_INSTANCE_ID}'/g' /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf && echo -e '\nInclude /usr/local/apache2/conf/opentemetry_agent.conf' >> /opt/opentelemetry-webserver/source-conf/httpd.conf"},
							Env: []corev1.EnvVar{
								{
									Name:  apacheAttributesEnvVar,
									Value: "\n#Load the Otel Webserver SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_common.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_resources.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_trace.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_otlp_recordable.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_ostream_span.so\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_exporter_otlp_grpc.so\n#Load the Otel ApacheModule SDK\nLoadFile /opt/opentelemetry-webserver/agent/sdk_lib/lib/libopentelemetry_webserver_sdk.so\n#Load the Apache Module. In this example for Apache 2.4\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Load the Apache Module. In this example for Apache 2.2\n#LoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel22.so\nLoadModule otel_apache_module /opt/opentelemetry-webserver/agent/WebServerModule/Apache/libmod_apache_otel.so\n#Attributes\nApacheModuleEnabled ON\nApacheModuleOtelExporterEndpoint https://collector:4318\nApacheModuleOtelSpanExporter otlp\nApacheModuleResolveBackends  ON\nApacheModuleServiceInstanceId <<SID-PLACEHOLDER>>\nApacheModuleServiceName app\nApacheModuleServiceNamespace ns\nApacheModuleTraceAsError  ON\n",
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
									Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
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

			pod := inj.inject(context.Background(), test.insts, testNamespace, test.pod, test.config)
			assert.Equal(t, test.expected, pod)
		})
	}
}

func TestInjectNginx(t *testing.T) {

	tests := []struct {
		name     string
		insts    languageInstrumentations
		pod      corev1.Pod
		expected corev1.Pod
		config   config.Config
	}{
		{
			name: "injection enabled, exporter set",
			insts: languageInstrumentations{
				Nginx: instrumentationWithContainers{
					Instrumentation: &v1alpha1.Instrumentation{
						Spec: v1alpha1.InstrumentationSpec{
							Nginx: v1alpha1.Nginx{
								Image: "img:1",
								Attrs: []corev1.EnvVar{{
									Name:  "NginxModuleOtelMaxQueueSize",
									Value: "4096",
								}},
							},
							Exporter: v1alpha1.Exporter{
								Endpoint: "http://otlp-endpoint:4317",
							},
						},
					},
					Containers: []string{"app"},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-nginx-6c44bcbdd",
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
					Name: "my-nginx-6c44bcbdd",
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
							Image:   "img:1",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{nginxSdkInitContainerTestCommand},
							Env: []corev1.EnvVar{
								{
									Name:  nginxAttributesEnvVar,
									Value: "NginxModuleEnabled ON;\nNginxModuleOtelExporterEndpoint http://otlp-endpoint:4317;\nNginxModuleOtelMaxQueueSize 4096;\nNginxModuleOtelSpanExporter otlp;\nNginxModuleResolveBackends ON;\nNginxModuleServiceInstanceId <<SID-PLACEHOLDER>>;\nNginxModuleServiceName my-nginx-6c44bcbdd;\nNginxModuleServiceNamespace ns;\nNginxModuleTraceAsError ON;\n",
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
							Name: "app",
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
									Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app",
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
			pod := inj.inject(context.Background(), test.insts, testNamespace, test.pod, test.config)
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
		Sdk: instrumentationWithContainers{Instrumentation: &inst, Containers: []string{"app"}},
	}

	inj := sdkInjector{
		logger: logr.Discard(),
	}
	config := config.New()
	pod := inj.inject(context.Background(), insts,
		testNamespace,
		corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "app:latest",
					},
				},
			},
		}, config)
	assert.Equal(t, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
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
							Value: "k8s.container.name=app,k8s.namespace.name=ns,k8s.node.name=$(OTEL_RESOURCE_ATTRIBUTES_NODE_NAME),k8s.pod.name=$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME),service.instance.id=ns.$(OTEL_RESOURCE_ATTRIBUTES_POD_NAME).app,service.version=latest",
						},
					},
				},
			},
		},
	}, pod)
}

func TestParentResourceLabels(t *testing.T) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-parent-resource-labels",
		},
	}
	err := k8sClient.Create(context.Background(), &ns)
	require.NoError(t, err)

	tests := []struct {
		name              string
		prepare           func()
		podObjectMeta     metav1.ObjectMeta
		expectedResources map[attribute.Key]string
	}{
		{
			name:              "from orphan pod",
			podObjectMeta:     metav1.ObjectMeta{},
			expectedResources: map[attribute.Key]string{},
		},
		{
			name: "from replicaset",
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       "my-rs",
						UID:        "my-rs-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SReplicaSetNameKey: "my-rs",
				semconv.K8SReplicaSetUIDKey:  "my-rs-uid",
			},
		},
		{
			name: "from deployment",
			prepare: func() {
				err := k8sClient.Create(context.Background(), &appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deploy-rs",
						Namespace: ns.Name,
						OwnerReferences: []metav1.OwnerReference{
							{ // from Deployment
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "my-deploy",
								UID:        "my-deploy-uid",
							},
						},
					},
					Spec: appsv1.ReplicaSetSpec{
						Replicas: ptr.To[int32](0),
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "my-deploy"}},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "my-deploy"}},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "con", Image: "img:1"}},
							},
						},
					},
				})
				require.NoError(t, err)
			},
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       "my-deploy-rs",
						UID:        "my-deploy-rs-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SReplicaSetNameKey: "my-deploy-rs",
				semconv.K8SReplicaSetUIDKey:  "my-deploy-rs-uid",
				semconv.K8SDeploymentNameKey: "my-deploy",
				semconv.K8SDeploymentUIDKey:  "my-deploy-uid",
			},
		},
		{
			name: "from job",
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "batch/v1",
						Kind:       "Job",
						Name:       "my-job",
						UID:        "my-job-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SJobNameKey: "my-job",
				semconv.K8SJobUIDKey:  "my-job-uid",
			},
		},
		{
			name: "from cronjob",
			prepare: func() {
				err := k8sClient.Create(context.Background(), &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-cronjob-job",
						Namespace: ns.Name,
						OwnerReferences: []metav1.OwnerReference{
							{ // from CronJob
								APIVersion: "batch/v1",
								Kind:       "CronJob",
								Name:       "my-cronjob",
								UID:        "my-cronjob-uid",
							},
						},
					},
					Spec: batchv1.JobSpec{
						Suspend: ptr.To[bool](true),
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyNever,
								Containers:    []corev1.Container{{Name: "con", Image: "img:1"}},
							},
						},
					},
				})
				require.NoError(t, err)
			},
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "batch/v1",
						Kind:       "Job",
						Name:       "my-cronjob-job",
						UID:        "my-cronjob-job-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SJobNameKey:     "my-cronjob-job",
				semconv.K8SJobUIDKey:      "my-cronjob-job-uid",
				semconv.K8SCronJobNameKey: "my-cronjob",
				semconv.K8SCronJobUIDKey:  "my-cronjob-uid",
			},
		},
		{
			name: "from statefulset",
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       "my-statefulset",
						UID:        "my-statefulset-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SStatefulSetNameKey: "my-statefulset",
				semconv.K8SStatefulSetUIDKey:  "my-statefulset-uid",
			},
		},
		{
			name: "from daemonset",
			podObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "DaemonSet",
						Name:       "my-daemonset",
						UID:        "my-daemonset-uid",
					},
				},
			},
			expectedResources: map[attribute.Key]string{
				semconv.K8SDaemonSetNameKey: "my-daemonset",
				semconv.K8SDaemonSetUIDKey:  "my-daemonset-uid",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			k8sResources := map[attribute.Key]string{}
			inj := sdkInjector{
				client: k8sClient,
				logger: logr.Discard(),
			}
			inj.addParentResourceLabels(context.Background(), true, ns, test.podObjectMeta, k8sResources)

			for k, v := range test.expectedResources {
				assert.Equal(t, v, k8sResources[k])
			}
		})
	}
}

func TestChooseServiceName(t *testing.T) {
	tests := []struct {
		name                           string
		resources                      map[string]string
		index                          int
		expectedServiceName            string
		useLabelsForResourceAttributes bool
		labelValue                     string
		annotationValue                string
	}{
		{
			name:                "first container",
			resources:           map[string]string{},
			index:               0,
			expectedServiceName: "1st",
		},
		{
			name:                "second container",
			resources:           map[string]string{},
			index:               1,
			expectedServiceName: "2nd",
		},
		{
			name: "from pod",
			resources: map[string]string{
				string(semconv.K8SPodNameKey): "my-pod",
			},
			index:               0,
			expectedServiceName: "my-pod",
		},
		{
			name: "from pod label - useLabelsForResourceAttributes=false",
			resources: map[string]string{
				string(semconv.K8SPodNameKey): "my-pod",
			},
			index:                          0,
			labelValue:                     "annotation",
			useLabelsForResourceAttributes: false,
			expectedServiceName:            "my-pod",
		},
		{
			name: "from pod label - useLabelsForResourceAttributes=true",
			resources: map[string]string{
				string(semconv.K8SPodNameKey): "my-pod",
			},
			index:                          0,
			labelValue:                     "label",
			useLabelsForResourceAttributes: true,
			expectedServiceName:            "label",
		},
		{
			name: "from pod annotation - useLabelsForResourceAttributes=false",
			resources: map[string]string{
				string(semconv.K8SPodNameKey): "my-pod",
			},
			index:                          0,
			annotationValue:                "annotation",
			labelValue:                     "label",
			useLabelsForResourceAttributes: false,
			expectedServiceName:            "annotation",
		},
		{
			name: "from pod annotation - useLabelsForResourceAttributes=true",
			resources: map[string]string{
				string(semconv.K8SPodNameKey): "my-pod",
			},
			index:                          0,
			annotationValue:                "annotation",
			labelValue:                     "label",
			useLabelsForResourceAttributes: true,
			expectedServiceName:            "annotation",
		},
		{
			name: "from replicaset",
			resources: map[string]string{
				string(semconv.K8SReplicaSetNameKey): "my-rs",
				string(semconv.K8SPodNameKey):        "my-rs-pod",
			},
			index:               0,
			expectedServiceName: "my-rs",
		},
		{
			name: "from deployment",
			resources: map[string]string{
				string(semconv.K8SDeploymentNameKey): "my-deploy",
				string(semconv.K8SReplicaSetNameKey): "my-deploy-rs",
				string(semconv.K8SPodNameKey):        "my-deploy-rs-pod",
			},
			index:               0,
			expectedServiceName: "my-deploy",
		},
		{
			name: "from cronjob",
			resources: map[string]string{
				string(semconv.K8SCronJobNameKey): "my-cronjob",
				string(semconv.K8SJobNameKey):     "my-cronjob-job",
				string(semconv.K8SPodNameKey):     "my-cronjob-job-pod",
			},
			index:               0,
			expectedServiceName: "my-cronjob",
		},
		{
			name: "from job",
			resources: map[string]string{
				string(semconv.K8SJobNameKey): "my-job",
				string(semconv.K8SPodNameKey): "my-job-pod",
			},
			index:               0,
			expectedServiceName: "my-job",
		},
		{
			name: "from statefulset",
			resources: map[string]string{
				string(semconv.K8SStatefulSetNameKey): "my-statefulset",
				string(semconv.K8SPodNameKey):         "my-statefulset-pod",
			},
			index:               0,
			expectedServiceName: "my-statefulset",
		},
		{
			name: "from daemonset",
			resources: map[string]string{
				string(semconv.K8SDaemonSetNameKey): "my-daemonset",
				string(semconv.K8SPodNameKey):       "my-daemonset-pod",
			},
			index:               0,
			expectedServiceName: "my-daemonset",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceName := chooseServiceName(corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": test.labelValue,
					},
					Annotations: map[string]string{
						"resource.opentelemetry.io/service.name": test.annotationValue,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "1st"},
						{Name: "2nd"},
					},
				},
			}, test.useLabelsForResourceAttributes, test.resources, test.index)

			assert.Equal(t, test.expectedServiceName, serviceName)
		})
	}
}
