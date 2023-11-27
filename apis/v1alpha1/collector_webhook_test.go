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

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var (
	testScheme *runtime.Scheme = scheme.Scheme
)

func TestOTELColDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)
	defaultCPUTarget := int32(90)

	if err := AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name     string
		otelcol  OpenTelemetryCollector
		expected OpenTelemetryCollector
	}{
		{
			name:    "all fields default",
			otelcol: OpenTelemetryCollector{},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					ManagementState: ManagementStateManaged,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					ManagementState: ManagementStateManaged,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ManagementState: ManagementStateUnmanaged,
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					ManagementState: ManagementStateUnmanaged,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &five,
						MinReplicas: &one,
					},
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					ManagementState: ManagementStateManaged,
					Autoscaler: &AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						MaxReplicas:          &five,
						MinReplicas:          &one,
					},
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &five,
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					ManagementState: ManagementStateManaged,
					Autoscaler: &AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						// webhook Default adds MaxReplicas to Autoscaler because
						// OpenTelemetryCollector.Spec.MaxReplicas is deprecated.
						MaxReplicas: &five,
						MinReplicas: &one,
					},
					MaxReplicas: &five,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					Ingress: Ingress{
						Type: IngressTypeRoute,
					},
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					ManagementState: ManagementStateManaged,
					Ingress: Ingress{
						Type: IngressTypeRoute,
						Route: OpenShiftRoute{
							Termination: TLSRouteTerminationTypeEdge,
						},
					},
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
						MinAvailable: &intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "10%",
						},
					},
				},
			},
			expected: OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeDeployment,
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					ManagementState: ManagementStateManaged,
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := &CollectorWebhook{
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
		otelcol          OpenTelemetryCollector
		expectedErr      string
		expectedWarnings []string
	}{
		{
			name:    "valid empty spec",
			otelcol: OpenTelemetryCollector{},
		},
		{
			name: "valid full spec",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeStatefulSet,
					MinReplicas:     &one,
					Replicas:        &three,
					MaxReplicas:     &five,
					UpgradeStrategy: "adhoc",
					TargetAllocator: OpenTelemetryTargetAllocator{
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
					Autoscaler: &AutoscalerSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:                 ModeSidecar,
					VolumeClaimTemplates: []v1.PersistentVolumeClaim{{}, {}},
				},
			},
			expectedErr: "does not support the attribute 'volumeClaimTemplates'",
		},
		{
			name: "invalid mode with tolerations",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:        ModeSidecar,
					Tolerations: []v1.Toleration{{}, {}},
				},
			},
			expectedErr: "does not support the attribute 'tolerations'",
		},
		{
			name: "invalid mode with target allocator",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: OpenTelemetryTargetAllocator{
						Enabled: true,
					},
				},
			},
			expectedErr: "does not support the target allocation deployment",
		},
		{
			name: "invalid target allocator config",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeStatefulSet,
					TargetAllocator: OpenTelemetryTargetAllocator{
						Enabled: true,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Prometheus configuration is incorrect",
		},
		{
			name: "invalid port name",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &zero,
				},
			},
			expectedErr:      "maxReplicas should be defined and one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid replicas, greater than max",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Replicas:    &five,
				},
			},
			expectedErr:      "replicas must not be greater than maxReplicas",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "invalid min replicas, greater than max",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					MinReplicas: &five,
				},
			},
			expectedErr:      "minReplicas must not be greater than maxReplicas",
			expectedWarnings: []string{"MaxReplicas is deprecated", "MinReplicas is deprecated"},
		},
		{
			name: "invalid min replicas, lesser than 1",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					MinReplicas: &zero,
				},
			},
			expectedErr:      "minReplicas should be one or more",
			expectedWarnings: []string{"MaxReplicas is deprecated", "MinReplicas is deprecated"},
		},
		{
			name: "invalid autoscaler scale down",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
						TargetCPUUtilization: &zero,
					},
				},
			},
			expectedErr:      "targetCPUUtilization should be greater than 0 and less than 100",
			expectedWarnings: []string{"MaxReplicas is deprecated"},
		},
		{
			name: "autoscaler minReplicas is less than maxReplicas",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &one,
						MinReplicas: &five,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas must not be greater than maxReplicas",
		},
		{
			name: "invalid autoscaler metric type",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
						Metrics: []MetricSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
						Metrics: []MetricSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					MaxReplicas: &three,
					Autoscaler: &AutoscalerSpec{
						Metrics: []MetricSpec{
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					Ingress: Ingress{
						Type: IngressTypeNginx,
					},
				},
			},
			expectedErr: fmt.Sprintf("Ingress can only be used in combination with the modes: %s, %s, %s",
				ModeDeployment, ModeDaemonSet, ModeStatefulSet,
			),
		},
		{
			name: "invalid mode with priorityClassName",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:              ModeSidecar,
					PriorityClassName: "test-class",
				},
			},
			expectedErr: "does not support the attribute 'priorityClassName'",
		},
		{
			name: "invalid mode with affinity",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						InitialDelaySeconds: &minusOne,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe InitialDelaySeconds configuration is incorrect",
		},
		{
			name: "invalid PeriodSeconds",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						PeriodSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe PeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid TimeoutSeconds",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						TimeoutSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TimeoutSeconds configuration is incorrect",
		},
		{
			name: "invalid SuccessThreshold",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						SuccessThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe SuccessThreshold configuration is incorrect",
		},
		{
			name: "invalid FailureThreshold",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						FailureThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe FailureThreshold configuration is incorrect",
		},
		{
			name: "invalid TerminationGracePeriodSeconds",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					LivenessProbe: &Probe{
						TerminationGracePeriodSeconds: &zero64,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TerminationGracePeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid AdditionalContainers",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
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
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Ingress: Ingress{
						RuleType: IngressRuleTypeSubdomain,
					},
				},
			},
			expectedErr: "a valid Ingress hostname has to be defined for subdomain ruleType",
		},
		{
			name: "invalid updateStrategy for Deployment mode",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
						Type: "RollingUpdate",
						RollingUpdate: &appsv1.RollingUpdateDaemonSet{
							MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
							MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Collector mode is set to deployment, which does not support the attribute 'updateStrategy'",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := &CollectorWebhook{
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
