package manifests

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
)

var (
	selectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "test.test",
	}
	basePolicy = corev1.ServiceInternalTrafficPolicyCluster
)

func TestBuildAll(t *testing.T) {
	var goodConfig = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [logging]
`
	one := int32(1)
	type args struct {
		params reconcileutil.Params
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
				params: reconcileutil.Params{
					Log: logr.Discard(),
					Instance: v1alpha1.OpenTelemetryCollector{
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
					Config: config.New(),
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
							"opentelemetry-operator-config/sha256": "baf97852b8beb44fb46a120f8c31873ded3129088e50cd6c69f3208ba60bd661",
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
									"opentelemetry-operator-config/sha256": "baf97852b8beb44fb46a120f8c31873ded3129088e50cd6c69f3208ba60bd661",
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
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildAll(tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
