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

package collectorwebhook

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var (
	testScheme *runtime.Scheme = scheme.Scheme
)

func TestOTELColDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)
	defaultCPUTarget := int32(90)

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name     string
		otelcol  v1alpha1.OpenTelemetryCollector
		expected v1alpha1.OpenTelemetryCollector
	}{
		{
			name:    "all fields default",
			otelcol: v1alpha1.OpenTelemetryCollector{},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					ManagementState: v1alpha1.ManagementStateManaged,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "provided values in spec",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					ManagementState: v1alpha1.ManagementStateManaged,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "doesn't override unmanaged",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					ManagementState: v1alpha1.ManagementStateUnmanaged,
					Mode:            v1alpha1.ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					ManagementState: v1alpha1.ManagementStateUnmanaged,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "Setting Autoscaler MaxReplicas",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1alpha1.AutoscalerSpec{
						MaxReplicas: &five,
						MinReplicas: &one,
					},
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					ManagementState: v1alpha1.ManagementStateManaged,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						MaxReplicas:          &five,
						MinReplicas:          &one,
					},
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "MaxReplicas but no Autoscale",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &five,
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					ManagementState: v1alpha1.ManagementStateManaged,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						// webhook Default adds MaxReplicas to Autoscaler because
						// OpenTelemetryCollector.Spec.MaxReplicas is deprecated.
						MaxReplicas: &five,
						MinReplicas: &one,
					},
					MaxReplicas: &five,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "Missing route termination",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeDeployment,
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
					},
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeDeployment,
					ManagementState: v1alpha1.ManagementStateManaged,
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.OpenShiftRoute{
							Termination: v1alpha1.TLSRouteTerminationTypeEdge,
						},
					},
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						},
					},
				},
			},
		},
		{
			name: "Defined PDB",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeDeployment,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MinAvailable: &intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "10%",
						},
					},
				},
			},
			expected: v1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					ManagementState: v1alpha1.ManagementStateManaged,
					PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
						MinAvailable: &intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "10%",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cvw := &Webhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
			}
			ctx := context.Background()
			err := cvw.Default(ctx, &test.otelcol)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.otelcol)
		})
	}
}

// TODO: a lot of these tests use .Spec.MaxReplicas and .Spec.MinReplicas. These fields are
// deprecated and moved to .Spec.Autoscaler. Fine to use these fields to test that old CRD is
// still supported but should eventually be updated.
func TestOTELColValidatingWebhook(t *testing.T) {
	minusOne := int32(-1)
	zero := int32(0)
	zero64 := int64(0)
	one := int32(1)
	three := int32(3)
	five := int32(5)

	tests := []struct { //nolint:govet
		name             string
		otelcol          v1alpha1.OpenTelemetryCollector
		expectedErr      string
		expectedWarnings []string
	}{
		{
			name:    "valid empty spec",
			otelcol: v1alpha1.OpenTelemetryCollector{},
		},
		{
			name: "valid full spec",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:            v1alpha1.ModeStatefulSet,
					MinReplicas:     &one,
					Replicas:        &three,
					MaxReplicas:     &five,
					UpgradeStrategy: "adhoc",
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled: true,
					},
					Config: `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  prometheus:
    config:
      scrape_configs:
        - job_name: otel-collector
          scrape_interval: 10s
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`,
					Ports: []v1.ServicePort{
						{
							Name: "port1",
							Port: 5555,
						},
						{
							Name:     "port2",
							Port:     5554,
							Protocol: v1.ProtocolUDP,
						},
					},
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleDown: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &three,
							},
							ScaleUp: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &five,
							},
						},
						TargetCPUUtilization: &five,
					},
				},
			},
		},
		{
			name: "invalid mode with volume claim templates",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:                 v1alpha1.ModeSidecar,
					VolumeClaimTemplates: []v1.PersistentVolumeClaim{{}, {}},
				},
			},
			expectedErr: "does not support the attribute 'volumeClaimTemplates'",
		},
		{
			name: "invalid mode with tolerations",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:        v1alpha1.ModeSidecar,
					Tolerations: []v1.Toleration{{}, {}},
				},
			},
			expectedErr: "does not support the attribute 'tolerations'",
		},
		{
			name: "invalid mode with target allocator",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeDeployment,
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled: true,
					},
				},
			},
			expectedErr: "does not support the target allocation deployment",
		},
		{
			name: "invalid target allocator config",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeStatefulSet,
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled: true,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Prometheus configuration is incorrect",
		},
		{
			name: "invalid port name",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ports: []v1.ServicePort{
						{
							// this port name contains a non alphanumeric character, which is invalid.
							Name:     "-test🦄port",
							Port:     12345,
							Protocol: v1.ProtocolTCP,
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port name, too long",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ports: []v1.ServicePort{
						{
							Name: "aaaabbbbccccdddd", // len: 16, too long
							Port: 5555,
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port num",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ports: []v1.ServicePort{
						{
							Name: "aaaabbbbccccddd", // len: 15
							// no port set means it's 0, which is invalid
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
		{
			name: "invalid max replicas",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &zero,
				},
			},
			expectedErr:      "maxReplicas should be defined and one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid replicas, greater than max",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Replicas:    &five,
				},
			},
			expectedErr:      "replicas must not be greater than maxReplicas",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid min replicas, greater than max",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					MinReplicas: &five,
				},
			},
			expectedErr:      "minReplicas must not be greater than maxReplicas",
			expectedWarnings: []string{"MaxReplicas is deprecated", "MinReplicas is deprecated"},
		},
		{
			name: "invalid min replicas, lesser than 1",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					MinReplicas: &zero,
				},
			},
			expectedErr:      "minReplicas should be one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated", "MinReplicas is deprecated"},
		},
		{
			name: "invalid autoscaler scale down",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleDown: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &zero,
							},
						},
					},
				},
			},
			expectedErr:      "scaleDown should be one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid autoscaler scale up",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleUp: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &zero,
							},
						},
					},
				},
			},
			expectedErr:      "scaleUp should be one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid autoscaler target cpu utilization",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						TargetCPUUtilization: &zero,
					},
				},
			},
			expectedErr:      "targetCPUUtilization should be greater than 0 and less than 100",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "autoscaler minReplicas is less than maxReplicas",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1alpha1.AutoscalerSpec{
						MaxReplicas: &one,
						MinReplicas: &five,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas must not be greater than maxReplicas",
		},
		{
			name: "invalid autoscaler metric type",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Metrics: []v1alpha1.MetricSpec{
							{
								Type: autoscalingv2.ResourceMetricSourceType,
							},
						},
					},
				},
			},
			expectedErr:      "the OpenTelemetry Spec autoscale configuration is incorrect, metric type unsupported. Expected metric of source type Pod",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid pod metric average value",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Metrics: []v1alpha1.MetricSpec{
							{
								Type: autoscalingv2.PodsMetricSourceType,
								Pods: &autoscalingv2.PodsMetricSource{
									Metric: autoscalingv2.MetricIdentifier{
										Name: "custom1",
									},
									Target: autoscalingv2.MetricTarget{
										Type:         autoscalingv2.AverageValueMetricType,
										AverageValue: resource.NewQuantity(int64(0), resource.DecimalSI),
									},
								},
							},
						},
					},
				},
			},
			expectedErr:      "the OpenTelemetry Spec autoscale configuration is incorrect, average value should be greater than 0",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "utilization target is not valid with pod metrics",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &v1alpha1.AutoscalerSpec{
						Metrics: []v1alpha1.MetricSpec{
							{
								Type: autoscalingv2.PodsMetricSourceType,
								Pods: &autoscalingv2.PodsMetricSource{
									Metric: autoscalingv2.MetricIdentifier{
										Name: "custom1",
									},
									Target: autoscalingv2.MetricTarget{
										Type:               autoscalingv2.UtilizationMetricType,
										AverageUtilization: &one,
									},
								},
							},
						},
					},
				},
			},
			expectedErr:      "the OpenTelemetry Spec autoscale configuration is incorrect, invalid pods target type",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid deployment mode incompabible with ingress settings",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeNginx,
					},
				},
			},
			expectedErr: fmt.Sprintf("Ingress can only be used in combination with the modes: %s, %s, %s",
				v1alpha1.ModeDeployment, v1alpha1.ModeDaemonSet, v1alpha1.ModeStatefulSet,
			),
		},
		{
			name: "invalid mode with priorityClassName",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode:              v1alpha1.ModeSidecar,
					PriorityClassName: "test-class",
				},
			},
			expectedErr: "does not support the attribute 'priorityClassName'",
		},
		{
			name: "invalid mode with affinity",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "node",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"test-node"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErr: "does not support the attribute 'affinity'",
		},
		{
			name: "invalid InitialDelaySeconds",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						InitialDelaySeconds: &minusOne,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe InitialDelaySeconds configuration is incorrect",
		},
		{
			name: "invalid PeriodSeconds",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						PeriodSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe PeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid TimeoutSeconds",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						TimeoutSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TimeoutSeconds configuration is incorrect",
		},
		{
			name: "invalid SuccessThreshold",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						SuccessThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe SuccessThreshold configuration is incorrect",
		},
		{
			name: "invalid FailureThreshold",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						FailureThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe FailureThreshold configuration is incorrect",
		},
		{
			name: "invalid TerminationGracePeriodSeconds",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1alpha1.Probe{
						TerminationGracePeriodSeconds: &zero64,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TerminationGracePeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid AdditionalContainers",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
					AdditionalContainers: []v1.Container{
						{
							Name: "test",
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Collector mode is set to sidecar, which does not support the attribute 'AdditionalContainers'",
		},
		{
			name: "missing ingress hostname for subdomain ruleType",
			otelcol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ingress: v1alpha1.Ingress{
						RuleType: v1alpha1.IngressRuleTypeSubdomain,
					},
				},
			},
			expectedErr: "a valid Ingress hostname has to be defined for subdomain ruleType",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cvw := &Webhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
			}
			ctx := context.Background()
			warnings, err := cvw.ValidateCreate(ctx, &test.otelcol)
			if test.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			if len(test.expectedWarnings) == 0 {
				assert.Empty(t, warnings, test.expectedWarnings)
			} else {
				assert.ElementsMatch(t, warnings, test.expectedWarnings)
			}
			assert.ErrorContains(t, err, test.expectedErr)
		})
	}
}
