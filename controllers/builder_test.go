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

package controllers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

var (
	selectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "test.test",
	}
	basePolicy     = corev1.ServiceInternalTrafficPolicyCluster
	pathTypePrefix = networkingv1.PathTypePrefix
)

var (
	opampbridgeSelectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
		"app.kubernetes.io/instance":   "test.test",
	}
)

func TestBuildCollector(t *testing.T) {
	var goodConfig = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
exporters:
  logging:
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [logging]
`
	one := int32(1)
	type args struct {
		instance v1alpha1.OpenTelemetryCollector
	}
	tests := []struct {
		name    string
		args    args
		want    []client.Object
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				instance: v1alpha1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Replicas: &one,
						Mode:     "deployment",
						Image:    "test",
						Config:   goodConfig,
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ingress",
			args: args{
				instance: v1alpha1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Replicas: &one,
						Mode:     "deployment",
						Image:    "test",
						Ingress: v1alpha1.Ingress{
							Type:     v1alpha1.IngressTypeNginx,
							Hostname: "example.com",
							Annotations: map[string]string{
								"something": "true",
							},
						},
						Config: goodConfig,
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ingress",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ingress",
						},
						Annotations: map[string]string{
							"something": "true",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "example.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/examplereceiver",
												PathType: &pathTypePrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "test-collector",
														Port: networkingv1.ServiceBackendPort{
															Name: "examplereceiver",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "specified service account case",
			args: args{
				instance: v1alpha1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Replicas:       &one,
						Mode:           "deployment",
						Image:          "test",
						Config:         goodConfig,
						ServiceAccount: "my-special-sa",
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "my-special-sa",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New(
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			)
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.args.instance,
			}
			got, err := BuildCollector(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)

		})
	}
}

func TestBuildAll_OpAMPBridge(t *testing.T) {
	one := int32(1)
	type args struct {
		instance v1alpha1.OpAMPBridge
	}
	tests := []struct {
		name    string
		args    args
		want    []client.Object
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				instance: v1alpha1.OpAMPBridge{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.OpAMPBridgeSpec{
						Replicas: &one,
						Image:    "test",
						Endpoint: "ws://opamp-server:4320/v1/opamp",
						Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
							v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
							v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
							v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
							v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
							v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
							v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
							v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
							v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
							v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
							v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
							v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
						},
						ComponentsAllowed: map[string][]string{"receivers": {"otlp"}, "processors": {"memory_limiter"}, "exporters": {"logging"}},
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-opamp-bridge",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-opamp-bridge",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: opampbridgeSelectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-opamp-bridge",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "opamp-bridge-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-opamp-bridge",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "remoteconfiguration.yaml",
														Path: "remoteconfiguration.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "opamp-bridge-container",
										Image: "test",
										Env: []corev1.EnvVar{
											{
												Name: "OTELCOL_NAMESPACE",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.namespace",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "opamp-bridge-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-opamp-bridge",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-opamp-bridge",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-opamp-bridge",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"remoteconfiguration.yaml": `capabilities:
  AcceptsOpAMPConnectionSettings: true
  AcceptsOtherConnectionSettings: true
  AcceptsRemoteConfig: true
  AcceptsRestartCommand: true
  ReportsEffectiveConfig: true
  ReportsHealth: true
  ReportsOwnLogs: true
  ReportsOwnMetrics: true
  ReportsOwnTraces: true
  ReportsRemoteConfig: true
  ReportsStatus: true
componentsAllowed:
  exporters:
  - logging
  processors:
  - memory_limiter
  receivers:
  - otlp
endpoint: ws://opamp-server:4320/v1/opamp
`},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-opamp-bridge",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-opamp-bridge",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-opamp-bridge",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-opamp-bridge",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "opamp-bridge",
								Port:       80,
								TargetPort: intstr.FromInt(8080),
							},
						},
						Selector: opampbridgeSelectorLabels,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New(
				config.WithOperatorOpAMPBridgeImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
				config.WithOperatorOpAMPBridgeImage("default-opamp-bridge"),
			)
			reconciler := NewOpAMPBridgeReconciler(OpAMPBridgeReconcilerParams{
				Log:    logr.Discard(),
				Config: cfg,
			})
			params := reconciler.getParams(tt.args.instance)
			got, err := BuildOpAMPBridge(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
