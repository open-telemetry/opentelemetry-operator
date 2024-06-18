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

package v1beta1

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	kubeTesting "k8s.io/client-go/testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

var (
	testScheme = scheme.Scheme
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		collector OpenTelemetryCollector
		warnings  []string
		err       string
	}{
		{
			name: "Test ",
			collector: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Config: Config{
						Processors: &AnyConfig{
							Object: map[string]interface{}{
								"batch": nil,
								"foo":   nil,
							},
						},
						Extensions: &AnyConfig{
							Object: map[string]interface{}{
								"foo": nil,
							},
						},
					},
				},
			},

			warnings: []string{
				"Collector config spec.config has null objects: extensions.foo:, processors.batch:, processors.foo:. For compatibility with other tooling, such as kustomize and kubectl edit, it is recommended to use empty objects e.g. batch: {}.",
			},
		},
	}
	for _, tt := range tests {
		webhook := CollectorWebhook{}
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			warnings, err := webhook.validate(context.Background(), &tt.collector)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				assert.Equal(t, tt.err, err.Error())
			}
			assert.ElementsMatch(t, tt.warnings, warnings)
		})
	}
}

func TestCollectorDefaultingWebhook(t *testing.T) {
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
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						ManagementState: ManagementStateManaged,
						Replicas:        &one,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					Mode:            ModeDeployment,
					UpgradeStrategy: UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "provided values in spec",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas: &five,
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
					Mode:            ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &five,
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
		},
		{
			name: "doesn't override unmanaged",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: ManagementStateUnmanaged,
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
					Mode:            ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &five,
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
					UpgradeStrategy: UpgradeStrategyAutomatic,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					Autoscaler: &AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						MaxReplicas:          &five,
						MinReplicas:          &one,
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						ManagementState: ManagementStateManaged,
						Replicas:        &one,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					Ingress: Ingress{
						Type: IngressTypeRoute,
						Route: OpenShiftRoute{
							Termination: TLSRouteTerminationTypeEdge,
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "Defined PDB for collector",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "Defined PDB for target allocator",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: TargetAllocatorAllocationStrategyConsistentHashing,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: TargetAllocatorAllocationStrategyConsistentHashing,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
		},
		{
			name: "Defined PDB for target allocator per-node",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: TargetAllocatorAllocationStrategyPerNode,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: TargetAllocatorAllocationStrategyPerNode,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
		},
		{
			name: "Undefined PDB for target allocator and consistent-hashing strategy",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: TargetAllocatorAllocationStrategyConsistentHashing,
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: TargetAllocatorAllocationStrategyConsistentHashing,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "Undefined PDB for target allocator and not consistent-hashing strategy",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: TargetAllocatorAllocationStrategyLeastWeighted,
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
					Mode: ModeDeployment,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: ManagementStateManaged,
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					UpgradeStrategy: UpgradeStrategyAutomatic,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: TargetAllocatorAllocationStrategyLeastWeighted,
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

var cfgYaml = `receivers:
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
`

func TestOTELColValidatingWebhook(t *testing.T) {
	minusOne := int32(-1)
	zero := int32(0)
	zero64 := int64(0)
	one := int32(1)
	three := int32(3)
	five := int32(5)

	cfg := Config{}
	err := yaml.Unmarshal([]byte(cfgYaml), &cfg)
	require.NoError(t, err)

	tests := []struct { //nolint:govet
		name             string
		otelcol          OpenTelemetryCollector
		expectedErr      string
		expectedWarnings []string
		shouldFailSar    bool
	}{
		{
			name:    "valid empty spec",
			otelcol: OpenTelemetryCollector{},
		},
		{
			name: "valid full spec",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeStatefulSet,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "port1",
									Port: 5555,
								},
							},
							{
								ServicePort: v1.ServicePort{
									Name:     "port2",
									Port:     5554,
									Protocol: v1.ProtocolUDP,
								},
							},
						},
					},
					Autoscaler: &AutoscalerSpec{
						MinReplicas: &one,
						MaxReplicas: &five,
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
					UpgradeStrategy: "adhoc",
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled: true,
					},
					Config: cfg,
				},
			},
		},
		{
			name:          "prom CR admissions warning",
			shouldFailSar: true, // force failure
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeStatefulSet,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "port1",
									Port: 5555,
								},
							},
							{
								ServicePort: v1.ServicePort{
									Name:     "port2",
									Port:     5554,
									Protocol: v1.ProtocolUDP,
								},
							},
						},
					},
					Autoscaler: &AutoscalerSpec{
						MinReplicas: &one,
						MaxReplicas: &five,
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
					UpgradeStrategy: "adhoc",
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:      true,
						PrometheusCR: TargetAllocatorPrometheusCR{Enabled: true},
					},
					Config: cfg,
				},
			},
			expectedWarnings: []string{
				"missing the following rules for monitoring.coreos.com/servicemonitors: [*]",
				"missing the following rules for monitoring.coreos.com/podmonitors: [*]",
				"missing the following rules for nodes/metrics: [get,list,watch]",
				"missing the following rules for services: [get,list,watch]",
				"missing the following rules for endpoints: [get,list,watch]",
				"missing the following rules for namespaces: [get,list,watch]",
				"missing the following rules for networking.k8s.io/ingresses: [get,list,watch]",
				"missing the following rules for nodes: [get,list,watch]",
				"missing the following rules for pods: [get,list,watch]",
				"missing the following rules for configmaps: [get]",
				"missing the following rules for discovery.k8s.io/endpointslices: [get,list,watch]",
				"missing the following rules for nonResourceURL: /metrics: [get]",
			},
		},
		{
			name:          "prom CR no admissions warning",
			shouldFailSar: false, // force SAR okay
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode:            ModeStatefulSet,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "port1",
									Port: 5555,
								},
							},
							{
								ServicePort: v1.ServicePort{
									Name:     "port2",
									Port:     5554,
									Protocol: v1.ProtocolUDP,
								},
							},
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
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:      true,
						PrometheusCR: TargetAllocatorPrometheusCR{Enabled: true},
					},
					Config: cfg,
				},
			},
		},
		{
			name: "invalid mode with volume claim templates",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					StatefulSetCommonFields: StatefulSetCommonFields{
						VolumeClaimTemplates: []v1.PersistentVolumeClaim{{}, {}},
					},
				},
			},
			expectedErr: "does not support the attribute 'volumeClaimTemplates'",
		},
		{
			name: "invalid mode with tolerations",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Tolerations: []v1.Toleration{{}, {}},
					},
				},
			},
			expectedErr: "does not support the attribute 'tolerations'",
		},
		{
			name: "invalid mode with target allocator",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDeployment,
					TargetAllocator: TargetAllocatorEmbedded{
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
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled: true,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Prometheus configuration is incorrect",
		},
		{
			name: "invalid target allocation strategy",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeDaemonSet,
					TargetAllocator: TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: TargetAllocatorAllocationStrategyLeastWeighted,
					},
				},
			},
			expectedErr: "mode is set to daemonset, which must be used with target allocation strategy per-node",
		},
		{
			name: "invalid port name",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									// this port name contains a non alphanumeric character, which is invalid.
									Name:     "-test🦄port",
									Port:     12345,
									Protocol: v1.ProtocolTCP,
								},
							},
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
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "aaaabbbbccccdddd", // len: 16, too long
									Port: 5555,
								},
							},
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
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Ports: []PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "aaaabbbbccccddd", // len: 15
									// no port set means it's 0, which is invalid
								},
							},
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
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &zero,
					},
				},
			},
			expectedErr: "maxReplicas should be defined and one or more",
		},
		{
			name: "invalid replicas, greater than max",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						Replicas: &five,
					},
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
					},
				},
			},
			expectedErr: "replicas must not be greater than maxReplicas",
		},
		{
			name: "invalid min replicas, greater than max",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
						MinReplicas: &five,
					},
				},
			},
			expectedErr: "minReplicas must not be greater than maxReplicas",
		},
		{
			name: "invalid min replicas, lesser than 1",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
						MinReplicas: &zero,
					},
				},
			},
			expectedErr: "minReplicas should be one or more",
		},
		{
			name: "invalid autoscaler scale down",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleDown: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &zero,
							},
						},
					},
				},
			},
			expectedErr: "scaleDown should be one or more",
		},
		{
			name: "invalid autoscaler scale up",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleUp: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &zero,
							},
						},
					},
				},
			},
			expectedErr: "scaleUp should be one or more",
		},
		{
			name: "invalid autoscaler target cpu utilization",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas:          &three,
						TargetCPUUtilization: &zero,
					},
				},
			},
			expectedErr: "targetCPUUtilization should be greater than 0 and less than 100",
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
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
						Metrics: []MetricSpec{
							{
								Type: autoscalingv2.ResourceMetricSourceType,
							},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, metric type unsupported. Expected metric of source type Pod",
		},
		{
			name: "invalid pod metric average value",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
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
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, average value should be greater than 0",
		},
		{
			name: "utilization target is not valid with pod metrics",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Autoscaler: &AutoscalerSpec{
						MaxReplicas: &three,
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
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, invalid pods target type",
		},
		{
			name: "invalid deployment mode incompabible with ingress settings",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					Ingress: Ingress{
						Type: IngressTypeIngress,
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
					Mode: ModeSidecar,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						PriorityClassName: "test-class",
					},
				},
			},
			expectedErr: "does not support the attribute 'priorityClassName'",
		},
		{
			name: "invalid mode with affinity",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
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
			name: "invalid InitialDelaySeconds readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						InitialDelaySeconds: &minusOne,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe InitialDelaySeconds configuration is incorrect",
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
			name: "invalid PeriodSeconds readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						PeriodSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe PeriodSeconds configuration is incorrect",
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
			name: "invalid TimeoutSeconds readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						TimeoutSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe TimeoutSeconds configuration is incorrect",
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
			name: "invalid SuccessThreshold readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						SuccessThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe SuccessThreshold configuration is incorrect",
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
			name: "invalid FailureThreshold readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						FailureThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe FailureThreshold configuration is incorrect",
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
			name: "invalid TerminationGracePeriodSeconds readiness",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					ReadinessProbe: &Probe{
						TerminationGracePeriodSeconds: &zero64,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe TerminationGracePeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid AdditionalContainers",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeSidecar,
					OpenTelemetryCommonFields: OpenTelemetryCommonFields{
						AdditionalContainers: []v1.Container{
							{
								Name: "test",
							},
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
					DaemonSetUpdateStrategy: appsv1.DaemonSetUpdateStrategy{
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
		{
			name: "invalid updateStrategy for Statefulset mode",
			otelcol: OpenTelemetryCollector{
				Spec: OpenTelemetryCollectorSpec{
					Mode: ModeStatefulSet,
					DeploymentUpdateStrategy: appsv1.DeploymentStrategy{
						Type: "RollingUpdate",
						RollingUpdate: &appsv1.RollingUpdateDeployment{
							MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
							MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Collector mode is set to statefulset, which does not support the attribute 'deploymentUpdateStrategy'",
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
				reviewer: getReviewer(test.shouldFailSar),
			}
			ctx := context.Background()
			warnings, err := cvw.ValidateCreate(ctx, &test.otelcol)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectedErr)
			}
			assert.Equal(t, len(test.expectedWarnings), len(warnings))
			assert.ElementsMatch(t, warnings, test.expectedWarnings)
		})
	}
}

func getReviewer(shouldFailSAR bool) *rbac.Reviewer {
	c := fake.NewSimpleClientset()
	c.PrependReactor("create", "subjectaccessreviews", func(action kubeTesting.Action) (handled bool, ret runtime.Object, err error) {
		// check our expectation here
		if !action.Matches("create", "subjectaccessreviews") {
			return false, nil, fmt.Errorf("must be a create for a SAR")
		}
		sar, ok := action.(kubeTesting.CreateAction).GetObject().DeepCopyObject().(*authv1.SubjectAccessReview)
		if !ok || sar == nil {
			return false, nil, fmt.Errorf("bad object")
		}
		sar.Status = authv1.SubjectAccessReviewStatus{
			Allowed: !shouldFailSAR,
			Denied:  shouldFailSAR,
		}
		return true, sar, nil
	})
	return rbac.NewReviewer(c)
}
