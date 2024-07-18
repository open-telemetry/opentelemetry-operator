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
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
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
