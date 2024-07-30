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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	go_yaml "gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
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

var (
	taSelectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		"app.kubernetes.io/instance":   "test.test",
		"app.kubernetes.io/name":       "test-targetallocator",
	}
)

func TestBuildCollector(t *testing.T) {
	var goodConfigYaml = `receivers:
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

	goodConfig := v1beta1.Config{}
	err := go_yaml.Unmarshal([]byte(goodConfigYaml), &goodConfig)
	require.NoError(t, err)

	goodConfigHash, _ := manifestutils.GetConfigMapSHA(goodConfig)
	goodConfigHash = goodConfigHash[:8]

	one := int32(1)
	type args struct {
		instance v1beta1.OpenTelemetryCollector
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
				instance: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							Image:    "test",
							Replicas: &one,
						},
						Mode:   "deployment",
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
						Annotations: map[string]string{},
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
									"opentelemetry-operator-config/sha256": "6f6f11da374b2c1e42fc78fbe55e2d9bcc2f5998ab63a631b49c478e8c0f6af8",
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
													Name: "test-collector-" + goodConfigHash,
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ServiceAccountName:    "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-" + goodConfigHash,
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{},
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  logging: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - logging\n      receivers:\n        - examplereceiver\n",
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
						Annotations: map[string]string{},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                      "opentelemetry-collector",
							"app.kubernetes.io/instance":                       "test.test",
							"app.kubernetes.io/managed-by":                     "opentelemetry-operator",
							"app.kubernetes.io/name":                           "test-collector",
							"app.kubernetes.io/part-of":                        "opentelemetry",
							"app.kubernetes.io/version":                        "latest",
							"operator.opentelemetry.io/collector-service-type": "base",
						},
						Annotations: map[string]string{},
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
							"operator.opentelemetry.io/collector-service-type":     "headless",
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
							"app.kubernetes.io/component":                            "opentelemetry-collector",
							"app.kubernetes.io/instance":                             "test.test",
							"app.kubernetes.io/managed-by":                           "opentelemetry-operator",
							"app.kubernetes.io/name":                                 "test-collector-monitoring",
							"app.kubernetes.io/part-of":                              "opentelemetry",
							"app.kubernetes.io/version":                              "latest",
							"operator.opentelemetry.io/collector-service-type":       "monitoring",
							"operator.opentelemetry.io/collector-monitoring-service": "Exists",
						},
						Annotations: map[string]string{},
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
				instance: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							Image:    "test",
							Replicas: &one,
						},
						Mode: "deployment",
						Ingress: v1beta1.Ingress{
							Type:     v1beta1.IngressTypeIngress,
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
						Annotations: map[string]string{},
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
									"opentelemetry-operator-config/sha256": "6f6f11da374b2c1e42fc78fbe55e2d9bcc2f5998ab63a631b49c478e8c0f6af8",
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
													Name: "test-collector-" + goodConfigHash,
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ServiceAccountName:    "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-" + goodConfigHash,
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{},
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  logging: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - logging\n      receivers:\n        - examplereceiver\n",
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
						Annotations: map[string]string{},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                      "opentelemetry-collector",
							"app.kubernetes.io/instance":                       "test.test",
							"app.kubernetes.io/managed-by":                     "opentelemetry-operator",
							"app.kubernetes.io/name":                           "test-collector",
							"app.kubernetes.io/part-of":                        "opentelemetry",
							"app.kubernetes.io/version":                        "latest",
							"operator.opentelemetry.io/collector-service-type": "base",
						},
						Annotations: map[string]string{},
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
							"operator.opentelemetry.io/collector-service-type":     "headless",
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
							"app.kubernetes.io/component":                            "opentelemetry-collector",
							"app.kubernetes.io/instance":                             "test.test",
							"app.kubernetes.io/managed-by":                           "opentelemetry-operator",
							"app.kubernetes.io/name":                                 "test-collector-monitoring",
							"app.kubernetes.io/part-of":                              "opentelemetry",
							"app.kubernetes.io/version":                              "latest",
							"operator.opentelemetry.io/collector-service-type":       "monitoring",
							"operator.opentelemetry.io/collector-monitoring-service": "Exists",
						},
						Annotations: map[string]string{},
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
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
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
				instance: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							Image:          "test",
							Replicas:       &one,
							ServiceAccount: "my-special-sa",
						},
						Mode:   "deployment",
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
						Annotations: map[string]string{},
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
									"opentelemetry-operator-config/sha256": "6f6f11da374b2c1e42fc78fbe55e2d9bcc2f5998ab63a631b49c478e8c0f6af8",
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
													Name: "test-collector-" + goodConfigHash,
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ServiceAccountName:    "my-special-sa",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-" + goodConfigHash,
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{},
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  logging: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - logging\n      receivers:\n        - examplereceiver\n",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                      "opentelemetry-collector",
							"app.kubernetes.io/instance":                       "test.test",
							"app.kubernetes.io/managed-by":                     "opentelemetry-operator",
							"app.kubernetes.io/name":                           "test-collector",
							"app.kubernetes.io/part-of":                        "opentelemetry",
							"app.kubernetes.io/version":                        "latest",
							"operator.opentelemetry.io/collector-service-type": "base",
						},
						Annotations: map[string]string{},
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
							"operator.opentelemetry.io/collector-service-type":     "headless",
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
							"app.kubernetes.io/component":                            "opentelemetry-collector",
							"app.kubernetes.io/instance":                             "test.test",
							"app.kubernetes.io/managed-by":                           "opentelemetry-operator",
							"app.kubernetes.io/name":                                 "test-collector-monitoring",
							"app.kubernetes.io/part-of":                              "opentelemetry",
							"app.kubernetes.io/version":                              "latest",
							"operator.opentelemetry.io/collector-service-type":       "monitoring",
							"operator.opentelemetry.io/collector-monitoring-service": "Exists",
						},
						Annotations: map[string]string{},
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
						Annotations: map[string]string{
							"opentelemetry-opampbridge-config/hash": "bd5cfc0df684966e25597a2847d5a3bae2c2b037d8bf10e7ea402ebe4d41c9f0",
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
								DNSConfig:          &corev1.PodDNSConfig{},
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

func TestBuildTargetAllocator(t *testing.T) {
	var goodConfigYaml = `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        relabel_configs:
        - source_labels: ['__meta_service_id']
          target_label: 'job'
          replacement: 'my_service_$$1'
        - source_labels: ['__meta_service_name']
          target_label: 'instance'
          replacement: '$1'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$$1_$2'
exporters:
  logging:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [logging]
`

	goodConfig := v1beta1.Config{}
	err := go_yaml.Unmarshal([]byte(goodConfigYaml), &goodConfig)
	require.NoError(t, err)

	goodConfigHash, _ := manifestutils.GetConfigMapSHA(goodConfig)
	goodConfigHash = goodConfigHash[:8]

	one := int32(1)
	type args struct {
		instance v1beta1.OpenTelemetryCollector
	}
	tests := []struct {
		name         string
		args         args
		want         []client.Object
		featuregates []string
		wantErr      bool
	}{
		{
			name: "base case",
			args: args{
				instance: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							Image:    "test",
							Replicas: &one,
						},
						Mode:   "statefulset",
						Config: goodConfig,
						TargetAllocator: v1beta1.TargetAllocatorEmbedded{
							Enabled:        true,
							FilterStrategy: "relabel-config",
							PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
								Enabled: true,
							},
						},
					},
				},
			},
			want: []client.Object{
				&appsv1.StatefulSet{
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
						Annotations: map[string]string{},
					},
					Spec: appsv1.StatefulSetSpec{
						ServiceName: "test-collector",
						Replicas:    &one,
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
									"opentelemetry-operator-config/sha256": "39cae697770f9d7e183e8fa9ba56043315b62e19c7231537870acfaaabc30a43",
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
													Name: "test-collector-" + goodConfigHash,
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
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
											{
												Name:  "SHARD",
												Value: "0",
											},
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ServiceAccountName:    "test-collector",
							},
						},
						PodManagementPolicy: "Parallel",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-" + goodConfigHash,
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{},
					},
					Data: map[string]string{
						"collector.yaml": "exporters:\n    logging: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - logging\n            receivers:\n                - prometheus\n",
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
						Annotations: map[string]string{},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                            "opentelemetry-collector",
							"app.kubernetes.io/instance":                             "test.test",
							"app.kubernetes.io/managed-by":                           "opentelemetry-operator",
							"app.kubernetes.io/name":                                 "test-collector-monitoring",
							"app.kubernetes.io/part-of":                              "opentelemetry",
							"app.kubernetes.io/version":                              "latest",
							"operator.opentelemetry.io/collector-service-type":       "monitoring",
							"operator.opentelemetry.io/collector-monitoring-service": "Exists",
						},
						Annotations: map[string]string{},
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
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"targetallocator.yaml": `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: test.test
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: example
    metric_relabel_configs:
    - replacement: $1_$2
      source_labels:
      - job
      target_label: job
    relabel_configs:
    - replacement: my_service_$1
      source_labels:
      - __meta_service_id
      target_label: job
    - replacement: $1
      source_labels:
      - __meta_service_name
      target_label: instance
filter_strategy: relabel-config
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  service_monitor_selector: null
`,
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: taSelectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-targetallocator",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-targetallocator",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-targetallocator-config/hash": "9d78d2ecfad18bad24dec7e9a825b4ce45657ecbb2e6b32845b585b7c15ea407",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "ta-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-targetallocator",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "targetallocator.yaml",
														Path: "targetallocator.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "ta-container",
										Image: "default-ta-allocator",
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
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												HostPort:      0,
												ContainerPort: 8080,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
										},
										LivenessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/livez",
													Port: intstr.FromInt(8080),
												},
											},
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/readyz",
													Port: intstr.FromInt(8080),
												},
											},
										},
									},
								},
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ShareProcessNamespace: ptr.To(false),
								ServiceAccountName:    "test-targetallocator",
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "targetallocation",
								Port: 80,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "http",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "enable metrics case",
			args: args{
				instance: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							Image:    "test",
							Replicas: &one,
						},
						Mode:   "statefulset",
						Config: goodConfig,
						TargetAllocator: v1beta1.TargetAllocatorEmbedded{
							Enabled: true,
							PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
								Enabled: true,
							},
							FilterStrategy: "relabel-config",
							Observability: v1beta1.ObservabilitySpec{
								Metrics: v1beta1.MetricsConfigSpec{
									EnableMetrics: true,
								},
							},
						},
					},
				},
			},
			want: []client.Object{
				&appsv1.StatefulSet{
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
						Annotations: map[string]string{},
					},
					Spec: appsv1.StatefulSetSpec{
						ServiceName: "test-collector",
						Replicas:    &one,
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
									"opentelemetry-operator-config/sha256": "39cae697770f9d7e183e8fa9ba56043315b62e19c7231537870acfaaabc30a43",
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
													Name: "test-collector-" + goodConfigHash,
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
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
											{
												Name:  "SHARD",
												Value: "0",
											},
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ServiceAccountName:    "test-collector",
							},
						},
						PodManagementPolicy: "Parallel",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-" + goodConfigHash,
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{},
					},
					Data: map[string]string{
						"collector.yaml": "exporters:\n    logging: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - logging\n            receivers:\n                - prometheus\n",
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
						Annotations: map[string]string{},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                            "opentelemetry-collector",
							"app.kubernetes.io/instance":                             "test.test",
							"app.kubernetes.io/managed-by":                           "opentelemetry-operator",
							"app.kubernetes.io/name":                                 "test-collector-monitoring",
							"app.kubernetes.io/part-of":                              "opentelemetry",
							"app.kubernetes.io/version":                              "latest",
							"operator.opentelemetry.io/collector-service-type":       "monitoring",
							"operator.opentelemetry.io/collector-monitoring-service": "Exists",
						},
						Annotations: map[string]string{},
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
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"targetallocator.yaml": `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: test.test
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: example
    metric_relabel_configs:
    - replacement: $1_$2
      source_labels:
      - job
      target_label: job
    relabel_configs:
    - replacement: my_service_$1
      source_labels:
      - __meta_service_id
      target_label: job
    - replacement: $1
      source_labels:
      - __meta_service_name
      target_label: instance
filter_strategy: relabel-config
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  service_monitor_selector: null
`,
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: taSelectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-targetallocator",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-targetallocator",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-targetallocator-config/hash": "9d78d2ecfad18bad24dec7e9a825b4ce45657ecbb2e6b32845b585b7c15ea407",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "ta-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-targetallocator",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "targetallocator.yaml",
														Path: "targetallocator.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "ta-container",
										Image: "default-ta-allocator",
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
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												HostPort:      0,
												ContainerPort: 8080,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
										},
										LivenessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/livez",
													Port: intstr.FromInt(8080),
												},
											},
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/readyz",
													Port: intstr.FromInt(8080),
												},
											},
										},
									},
								},
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
								ShareProcessNamespace: ptr.To(false),
								ServiceAccountName:    "test-targetallocator",
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "targetallocation",
								Port: 80,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "http",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
				&monitoringv1.ServiceMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						Endpoints: []monitoringv1.Endpoint{
							{Port: "targetallocation"},
						},
						Selector: v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						NamespaceSelector: monitoringv1.NamespaceSelector{
							MatchNames: []string{"test"},
						},
					},
				},
			},
			wantErr:      false,
			featuregates: []string{},
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
			targetAllocator, err := collector.TargetAllocator(params)
			require.NoError(t, err)
			params.TargetAllocator = targetAllocator
			if len(tt.featuregates) > 0 {
				fg := strings.Join(tt.featuregates, ",")
				flagset := featuregate.Flags(colfeaturegate.GlobalRegistry())
				if err = flagset.Set(featuregate.FeatureGatesFlag, fg); err != nil {
					t.Errorf("featuregate setting error = %v", err)
					return
				}
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
