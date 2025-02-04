// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	go_yaml "gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
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
  debug:
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [debug]
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
									"opentelemetry-operator-config/sha256": "2d266e55025628659355f1271b689d6fb53648ef6cd5595831f5835d18e59a25",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  debug: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - debug\n      receivers:\n        - examplereceiver\n",
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
									"opentelemetry-operator-config/sha256": "2d266e55025628659355f1271b689d6fb53648ef6cd5595831f5835d18e59a25",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  debug: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - debug\n      receivers:\n        - examplereceiver\n",
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
									"opentelemetry-operator-config/sha256": "2d266e55025628659355f1271b689d6fb53648ef6cd5595831f5835d18e59a25",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: 0.0.0.0:12345\nexporters:\n  debug: null\nservice:\n  pipelines:\n    metrics:\n      exporters:\n        - debug\n      receivers:\n        - examplereceiver\n",
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
						ComponentsAllowed: map[string][]string{"receivers": {"otlp"}, "processors": {"memory_limiter"}, "exporters": {"debug"}},
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
							"opentelemetry-opampbridge-config/hash": "05e1dc681267a9bc28fc2877ab464a98b9bd043843f14ffc0b4a394b5c86ba9f",
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
  - debug
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

func TestBuildCollectorTargetAllocatorResources(t *testing.T) {
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
  debug:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [debug]
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
		featuregates []*colfeaturegate.Gate
		wantErr      bool
		opts         []config.Option
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
							Enabled:            true,
							FilterStrategy:     "relabel-config",
							AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
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
									"opentelemetry-operator-config/sha256": "42773025f65feaf30df59a306a9e38f1aaabe94c8310983beaddb7f648d699b0",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "exporters:\n    debug: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - debug\n            receivers:\n                - prometheus\n",
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
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
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
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
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
							AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
							FilterStrategy:     "relabel-config",
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
									"opentelemetry-operator-config/sha256": "42773025f65feaf30df59a306a9e38f1aaabe94c8310983beaddb7f648d699b0",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "exporters:\n    debug: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - debug\n            receivers:\n                - prometheus\n",
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
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
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
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
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
			wantErr: false,
		},
		{
			name: "target allocator mtls enabled",
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
							Enabled:            true,
							FilterStrategy:     "relabel-config",
							AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
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
									"opentelemetry-operator-config/sha256": "42773025f65feaf30df59a306a9e38f1aaabe94c8310983beaddb7f648d699b0",
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
									{
										Name: "test-ta-client-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-ta-client-cert",
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
											{
												Name:      "test-ta-client-cert",
												MountPath: "/tls",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "exporters:\n    debug: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: https://test-targetallocator:443\n            interval: 30s\n            tls:\n                ca_file: /tls/ca.crt\n                cert_file: /tls/tls.crt\n                key_file: /tls/tls.key\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - debug\n            receivers:\n                - prometheus\n",
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
https:
  ca_file_path: /tls/ca.crt
  enabled: true
  listen_addr: :8443
  tls_cert_file_path: /tls/tls.crt
  tls_key_file_path: /tls/tls.key
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "3e2818ab54d866289de7837779e86e9c95803c43c0c4b58b25123e809ae9b771",
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
									{
										Name: "test-ta-server-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-ta-server-cert",
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
											{
												Name:          "https",
												HostPort:      0,
												ContainerPort: 8443,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
											{
												Name:      "test-ta-server-cert",
												MountPath: "/tls",
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
							{
								Name: "targetallocation-https",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "https",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "3e2818ab54d866289de7837779e86e9c95803c43c0c4b58b25123e809ae9b771",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
				&cmv1.Issuer{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-self-signed-issuer",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-self-signed-issuer",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.IssuerSpec{
						IssuerConfig: cmv1.IssuerConfig{
							SelfSigned: &cmv1.SelfSignedIssuer{
								CRLDistributionPoints: nil,
							},
						},
					},
				},
				&cmv1.Certificate{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ca-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ca-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						CommonName: "test-ca-cert",
						IsCA:       true,
						SecretName: "test-ca-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-self-signed-issuer",
							Kind: "Issuer",
						},
					},
				},
				&cmv1.Issuer{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ca-issuer",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ca-issuer",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.IssuerSpec{
						IssuerConfig: cmv1.IssuerConfig{
							CA: &cmv1.CAIssuer{
								SecretName: "test-ca-cert",
							},
						},
					},
				},
				&cmv1.Certificate{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ta-server-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ta-server-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						DNSNames: []string{
							"test-targetallocator",
							"test-targetallocator.test.svc",
							"test-targetallocator.test.svc.cluster.local",
						},
						SecretName: "test-ta-server-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-ca-issuer",
							Kind: "Issuer",
						},
						Usages: []cmv1.KeyUsage{
							"client auth",
							"server auth",
						},
					},
				},
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ta-client-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ta-client-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						DNSNames: []string{
							"test-targetallocator",
							"test-targetallocator.test.svc",
							"test-targetallocator.test.svc.cluster.local",
						},
						SecretName: "test-ta-client-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-ca-issuer",
							Kind: "Issuer",
						},
						Usages: []cmv1.KeyUsage{
							"client auth",
							"server auth",
						},
					},
				},
			},
			wantErr: false,
			opts: []config.Option{
				config.WithCertManagerAvailability(certmanager.Available),
			},
			featuregates: []*colfeaturegate.Gate{featuregate.EnableTargetAllocatorMTLS},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []config.Option{
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			}
			opts = append(opts, tt.opts...)
			cfg := config.New(
				opts...,
			)
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.args.instance,
			}
			targetAllocator, err := collector.TargetAllocator(params)
			require.NoError(t, err)
			params.TargetAllocator = targetAllocator
			registry := colfeaturegate.GlobalRegistry()
			for _, gate := range tt.featuregates {
				current := gate.IsEnabled()
				require.False(t, current, "only enable gates which are disabled by default")
				if setErr := registry.Set(gate.ID(), true); setErr != nil {
					require.NoError(t, setErr)
					return
				}
				t.Cleanup(func() {
					setErr := registry.Set(gate.ID(), current)
					require.NoError(t, setErr)
				})
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

func TestBuildCollectorTargetAllocatorCR(t *testing.T) {
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
  debug:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [debug]
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
		featuregates []*colfeaturegate.Gate
		wantErr      bool
		opts         []config.Option
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
							Enabled:            true,
							FilterStrategy:     "relabel-config",
							AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
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
									"opentelemetry-operator-config/sha256": "42773025f65feaf30df59a306a9e38f1aaabe94c8310983beaddb7f648d699b0",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "exporters:\n    debug: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - debug\n            receivers:\n                - prometheus\n",
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
				&v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy:     v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
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
									"opentelemetry-operator-config/sha256": "42773025f65feaf30df59a306a9e38f1aaabe94c8310983beaddb7f648d699b0",
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
				&policyV1.PodDisruptionBudget{
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
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/version":    "latest",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
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
						"collector.yaml": "exporters:\n    debug: null\nreceivers:\n    prometheus:\n        config: {}\n        target_allocator:\n            collector_id: ${POD_NAME}\n            endpoint: http://test-targetallocator:80\n            interval: 30s\nservice:\n    pipelines:\n        metrics:\n            exporters:\n                - debug\n            receivers:\n                - prometheus\n",
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
				&v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy: v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
						Observability: v1beta1.ObservabilitySpec{
							Metrics: v1beta1.MetricsConfigSpec{
								EnableMetrics: true,
							},
						},
					},
				},
			},
			wantErr:      false,
			featuregates: []*colfeaturegate.Gate{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []config.Option{
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			}
			opts = append(opts, tt.opts...)
			cfg := config.New(
				opts...,
			)
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.args.instance,
			}
			targetAllocator, err := collector.TargetAllocator(params)
			require.NoError(t, err)
			params.TargetAllocator = targetAllocator
			featuregates := []*colfeaturegate.Gate{featuregate.CollectorUsesTargetAllocatorCR}
			featuregates = append(featuregates, tt.featuregates...)
			registry := colfeaturegate.GlobalRegistry()
			for _, gate := range featuregates {
				current := gate.IsEnabled()
				require.False(t, current, "only enable gates which are disabled by default")
				if setErr := registry.Set(gate.ID(), true); setErr != nil {
					require.NoError(t, setErr)
					return
				}
				t.Cleanup(func() {
					setErr := registry.Set(gate.ID(), current)
					require.NoError(t, setErr)
				})
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

func TestBuildTargetAllocator(t *testing.T) {
	type args struct {
		instance  v1alpha1.TargetAllocator
		collector *v1beta1.OpenTelemetryCollector
	}
	tests := []struct {
		name         string
		args         args
		want         []client.Object
		featuregates []*colfeaturegate.Gate
		wantErr      bool
		opts         []config.Option
	}{
		{
			name: "base case",
			args: args{
				instance: v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy: v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						ScrapeConfigs: []v1beta1.AnyConfig{
							{Object: map[string]any{
								"job_name": "example",
								"metric_relabel_configs": []any{
									map[string]any{
										"replacement":   "$1_$2",
										"source_labels": []any{"job"},
										"target_label":  "job",
									},
								},
								"relabel_configs": []any{
									map[string]any{
										"replacement":   "my_service_$1",
										"source_labels": []any{"__meta_service_id"},
										"target_label":  "job",
									},
									map[string]any{
										"replacement":   "$1",
										"source_labels": []any{"__meta_service_name"},
										"target_label":  "instance",
									},
								},
							}},
						},
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
					},
				},
			},
			want: []client.Object{
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
collector_selector: null
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
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "f80c054419fe2f9030368557da143e200c70772d1d5f1be50ed55ae960b4b17d",
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
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "f80c054419fe2f9030368557da143e200c70772d1d5f1be50ed55ae960b4b17d",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "enable metrics case",
			args: args{
				instance: v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy: v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						ScrapeConfigs: []v1beta1.AnyConfig{
							{Object: map[string]any{
								"job_name": "example",
								"metric_relabel_configs": []any{
									map[string]any{
										"replacement":   "$1_$2",
										"source_labels": []any{"job"},
										"target_label":  "job",
									},
								},
								"relabel_configs": []any{
									map[string]any{
										"replacement":   "my_service_$1",
										"source_labels": []any{"__meta_service_id"},
										"target_label":  "job",
									},
									map[string]any{
										"replacement":   "$1",
										"source_labels": []any{"__meta_service_name"},
										"target_label":  "instance",
									},
								},
							}},
						},
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
						Observability: v1beta1.ObservabilitySpec{
							Metrics: v1beta1.MetricsConfigSpec{
								EnableMetrics: true,
							},
						},
					},
				},
			},
			want: []client.Object{
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
collector_selector: null
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
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "f80c054419fe2f9030368557da143e200c70772d1d5f1be50ed55ae960b4b17d",
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
								ShareProcessNamespace: ptr.To(false),
								DNSPolicy:             "ClusterFirst",
								DNSConfig:             &corev1.PodDNSConfig{},
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
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "f80c054419fe2f9030368557da143e200c70772d1d5f1be50ed55ae960b4b17d",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
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
			wantErr: false,
		},
		{
			name: "collector present",
			args: args{
				instance: v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy: v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
					},
				},
				collector: &v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Receivers: v1beta1.AnyConfig{
								Object: map[string]any{
									"prometheus": map[string]any{
										"config": map[string]any{
											"scrape_configs": []any{
												map[string]any{
													"job_name": "example",
													"metric_relabel_configs": []any{
														map[string]any{
															"replacement":   "$1_$2",
															"source_labels": []any{"job"},
															"target_label":  "job",
														},
													},
													"relabel_configs": []any{
														map[string]any{
															"replacement":   "my_service_$1",
															"source_labels": []any{"__meta_service_id"},
															"target_label":  "job",
														},
														map[string]any{
															"replacement":   "$1",
															"source_labels": []any{"__meta_service_name"},
															"target_label":  "instance",
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
			want: []client.Object{
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
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
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
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "286a5a4e7ec6d2ce652a4ce23e135c10053b4c87fd080242daa5bf21dcd5a337",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mtls",
			args: args{
				instance: v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
						Labels:    nil,
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						FilterStrategy: v1beta1.TargetAllocatorFilterStrategyRelabelConfig,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
							Enabled: true,
						},
					},
				},
				collector: &v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Receivers: v1beta1.AnyConfig{
								Object: map[string]any{
									"prometheus": map[string]any{
										"config": map[string]any{
											"scrape_configs": []any{
												map[string]any{
													"job_name": "example",
													"metric_relabel_configs": []any{
														map[string]any{
															"replacement":   "$1_$2",
															"source_labels": []any{"job"},
															"target_label":  "job",
														},
													},
													"relabel_configs": []any{
														map[string]any{
															"replacement":   "my_service_$1",
															"source_labels": []any{"__meta_service_id"},
															"target_label":  "job",
														},
														map[string]any{
															"replacement":   "$1",
															"source_labels": []any{"__meta_service_name"},
															"target_label":  "instance",
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
			want: []client.Object{
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
https:
  ca_file_path: /tls/ca.crt
  enabled: true
  listen_addr: :8443
  tls_cert_file_path: /tls/tls.crt
  tls_key_file_path: /tls/tls.key
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  probe_selector: null
  scrape_config_selector: null
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
									"opentelemetry-targetallocator-config/hash": "3e2818ab54d866289de7837779e86e9c95803c43c0c4b58b25123e809ae9b771",
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
									{
										Name: "test-ta-server-cert",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{
												SecretName: "test-ta-server-cert",
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
											{
												Name:          "https",
												HostPort:      0,
												ContainerPort: 8443,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
											{
												Name:      "test-ta-server-cert",
												MountPath: "/tls",
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
							{
								Name: "targetallocation-https",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "https",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
				&policyV1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{},
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
						Annotations: map[string]string{
							"opentelemetry-targetallocator-config/hash": "3e2818ab54d866289de7837779e86e9c95803c43c0c4b58b25123e809ae9b771",
						},
					},
					Spec: policyV1.PodDisruptionBudgetSpec{
						Selector: &v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
				&cmv1.Issuer{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-self-signed-issuer",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-self-signed-issuer",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.IssuerSpec{
						IssuerConfig: cmv1.IssuerConfig{
							SelfSigned: &cmv1.SelfSignedIssuer{
								CRLDistributionPoints: nil,
							},
						},
					},
				},
				&cmv1.Certificate{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ca-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ca-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						CommonName: "test-ca-cert",
						IsCA:       true,
						SecretName: "test-ca-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-self-signed-issuer",
							Kind: "Issuer",
						},
					},
				},
				&cmv1.Issuer{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ca-issuer",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ca-issuer",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.IssuerSpec{
						IssuerConfig: cmv1.IssuerConfig{
							CA: &cmv1.CAIssuer{
								SecretName: "test-ca-cert",
							},
						},
					},
				},
				&cmv1.Certificate{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ta-server-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ta-server-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						DNSNames: []string{
							"test-targetallocator",
							"test-targetallocator.test.svc",
							"test-targetallocator.test.svc.cluster.local",
						},
						SecretName: "test-ta-server-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-ca-issuer",
							Kind: "Issuer",
						},
						Usages: []cmv1.KeyUsage{
							"client auth",
							"server auth",
						},
					},
				},
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ta-client-cert",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ta-client-cert",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
					Spec: cmv1.CertificateSpec{
						Subject: &cmv1.X509Subject{
							OrganizationalUnits: []string{"opentelemetry-operator"},
						},
						DNSNames: []string{
							"test-targetallocator",
							"test-targetallocator.test.svc",
							"test-targetallocator.test.svc.cluster.local",
						},
						SecretName: "test-ta-client-cert",
						IssuerRef: cmmetav1.ObjectReference{
							Name: "test-ca-issuer",
							Kind: "Issuer",
						},
						Usages: []cmv1.KeyUsage{
							"client auth",
							"server auth",
						},
					},
				},
			},
			wantErr: false,
			opts: []config.Option{
				config.WithCertManagerAvailability(certmanager.Available),
			},
			featuregates: []*colfeaturegate.Gate{featuregate.EnableTargetAllocatorMTLS},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []config.Option{
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			}
			opts = append(opts, tt.opts...)
			cfg := config.New(
				opts...,
			)
			params := targetallocator.Params{
				Log:             logr.Discard(),
				Config:          cfg,
				TargetAllocator: tt.args.instance,
				Collector:       tt.args.collector,
			}
			registry := colfeaturegate.GlobalRegistry()
			for _, gate := range tt.featuregates {
				current := gate.IsEnabled()
				require.False(t, current, "only enable gates which are disabled by default")
				if err := registry.Set(gate.ID(), true); err != nil {
					require.NoError(t, err)
					return
				}
				t.Cleanup(func() {
					err := registry.Set(gate.ID(), current)
					require.NoError(t, err)
				})
			}
			got, err := BuildTargetAllocator(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)

		})
	}
}
