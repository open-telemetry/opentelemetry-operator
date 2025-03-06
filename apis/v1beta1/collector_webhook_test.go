// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"fmt"
	"math"
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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	collectorManifests "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

var (
	testScheme = scheme.Scheme
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		collector     v1beta1.OpenTelemetryCollector
		warnings      []string
		err           string
		shouldFailSar bool
	}{
		{
			name: "Test ",
			collector: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Processors: &v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"batch": nil,
								"foo":   nil,
							},
						},
						Extensions: &v1beta1.AnyConfig{
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

	bv := func(_ context.Context, collector v1beta1.OpenTelemetryCollector) admission.Warnings {
		var warnings admission.Warnings
		cfg := config.New(
			config.WithCollectorImage("default-collector"),
			config.WithTargetAllocatorImage("default-ta-allocator"),
		)
		params := manifests.Params{
			Log:     logr.Discard(),
			Config:  cfg,
			OtelCol: collector,
		}
		_, err := collectorManifests.Build(params)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		return nil
	}

	for _, tt := range tests {
		test := tt
		webhook := v1beta1.NewCollectorWebhook(
			logr.Discard(),
			testScheme,
			config.New(
				config.WithCollectorImage("collector:v0.0.0"),
				config.WithTargetAllocatorImage("ta:v0.0.0"),
			),
			getReviewer(test.shouldFailSar),
			nil,
			bv,
			nil,
		)
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			warnings, err := webhook.Validate(context.Background(), &tt.collector)
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

	if err := v1beta1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name          string
		otelcol       v1beta1.OpenTelemetryCollector
		expected      v1beta1.OpenTelemetryCollector
		shouldFailSar bool
	}{
		{
			name: "update config defaults",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: func() v1beta1.Config {
						const input = `{"receivers":{"otlp":{"protocols":{"grpc":null,"http":null}}},"exporters":{"debug":null},"service":{"pipelines":{"traces":{"receivers":["otlp"],"exporters":["debug"]}}}}`
						var cfg v1beta1.Config
						require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))
						return cfg
					}(),
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						ManagementState: v1beta1.ManagementStateManaged,
						Replicas:        &one,
					},
					Mode:            v1beta1.ModeDeployment,
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					Config: func() v1beta1.Config {
						const input = `{"receivers":{"otlp":{"protocols":{"grpc":{"endpoint":"0.0.0.0:4317"},"http":{"endpoint":"0.0.0.0:4318"}}}},"exporters":{"debug":null},"service":{"telemetry":{"metrics":{"address":"0.0.0.0:8888"}},"pipelines":{"traces":{"receivers":["otlp"],"exporters":["debug"]}}}}`
						var cfg v1beta1.Config
						require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))
						return cfg
					}(),
				},
			},
		},
		{
			name: "update config defaults, leave other fields alone",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: func() v1beta1.Config {
						const input = `{"receivers":{"otlp":{"protocols":{"grpc":{"headers":{"example":"another"}},"http":{"endpoint":"0.0.0.0:4000"}}}},"exporters":{"debug":null},"service":{"telemetry":{"metrics":{"address":"1.2.3.4:7654"}},"pipelines":{"traces":{"receivers":["otlp"],"exporters":["debug"]}}}}`
						var cfg v1beta1.Config
						require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))
						return cfg
					}(),
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						ManagementState: v1beta1.ManagementStateManaged,
						Replicas:        &one,
					},
					Mode:            v1beta1.ModeDeployment,
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					Config: func() v1beta1.Config {
						const input = `{"receivers":{"otlp":{"protocols":{"grpc":{"endpoint":"0.0.0.0:4317","headers":{"example":"another"}},"http":{"endpoint":"0.0.0.0:4000"}}}},"exporters":{"debug":null},"service":{"telemetry":{"metrics":{"address":"1.2.3.4:7654"}},"pipelines":{"traces":{"receivers":["otlp"],"exporters":["debug"]}}}}`
						var cfg v1beta1.Config
						require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))
						return cfg
					}(),
				},
			},
		},
		{
			name:    "all fields default",
			otelcol: v1beta1.OpenTelemetryCollector{},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						ManagementState: v1beta1.ManagementStateManaged,
						Replicas:        &one,
					},
					Mode:            v1beta1.ModeDeployment,
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "provided values in spec",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &five,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: v1beta1.ManagementStateManaged,
					},
				},
			},
		},
		{
			name: "doesn't override unmanaged",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: v1beta1.ManagementStateUnmanaged,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeSidecar,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: v1beta1.ManagementStateUnmanaged,
					},
				},
			},
		},
		{
			name: "Setting Autoscaler MaxReplicas",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &five,
						MinReplicas: &one,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeDeployment,
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
					},
					Autoscaler: &v1beta1.AutoscalerSpec{
						TargetCPUUtilization: &defaultCPUTarget,
						MaxReplicas:          &five,
						MinReplicas:          &one,
					},
				},
			},
		},
		{
			name: "Missing route termination",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressTypeRoute,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						ManagementState: v1beta1.ManagementStateManaged,
						Replicas:        &one,
					},
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressTypeRoute,
						Route: v1beta1.OpenShiftRoute{
							Termination: v1beta1.TLSRouteTerminationTypeEdge,
						},
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "Defined PDB for collector",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "Defined PDB for target allocator",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyPerNode,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyPerNode,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
					},
				},
			},
		},
		{
			name: "Undefined PDB for target allocator and not consistent-hashing strategy",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
					},
				},
			},
			expected: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &one,
						ManagementState: v1beta1.ManagementStateManaged,
					},
					UpgradeStrategy: v1beta1.UpgradeStrategyAutomatic,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						Replicas:           &one,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
					},
				},
			},
		},
	}

	bv := func(_ context.Context, collector v1beta1.OpenTelemetryCollector) admission.Warnings {
		var warnings admission.Warnings
		cfg := config.New(
			config.WithCollectorImage("default-collector"),
			config.WithTargetAllocatorImage("default-ta-allocator"),
		)
		params := manifests.Params{
			Log:     logr.Discard(),
			Config:  cfg,
			OtelCol: collector,
		}
		_, err := collectorManifests.Build(params)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		return nil
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := v1beta1.NewCollectorWebhook(
				logr.Discard(),
				testScheme,
				config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
				getReviewer(test.shouldFailSar),
				nil,
				bv,
				nil,
			)
			ctx := context.Background()
			err := cvw.Default(ctx, &test.otelcol)
			if test.expected.Spec.Config.Service.Telemetry == nil {
				assert.NoError(t, test.expected.Spec.Config.Service.ApplyDefaults(logr.Discard()), "could not apply defaults")
			}
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
	maxInt := int32(math.MaxInt32)

	cfg := v1beta1.Config{
		Service: v1beta1.Service{
			Telemetry: &v1beta1.AnyConfig{
				Object: map[string]interface{}{
					"metrics": map[string]interface{}{
						"address": "${env:POD_ID}:8888",
					},
				},
			},
		},
	}
	err := yaml.Unmarshal([]byte(cfgYaml), &cfg)
	require.NoError(t, err)

	tests := []struct { //nolint:govet
		name             string
		otelcol          v1beta1.OpenTelemetryCollector
		expectedErr      string
		expectedWarnings []string
		shouldFailSar    bool
	}{
		{
			name:    "valid empty spec",
			otelcol: v1beta1.OpenTelemetryCollector{},
		},
		{
			name: "valid full spec",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeStatefulSet,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []v1beta1.PortsSpec{
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
					Autoscaler: &v1beta1.AutoscalerSpec{
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
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled: true,
					},
					Config: cfg,
				},
			},
		},
		{
			name:          "prom CR admissions warning",
			shouldFailSar: true, // force failure
			otelcol: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "adm-warning",
					Namespace: "test-ns",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeStatefulSet,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []v1beta1.PortsSpec{
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
					Autoscaler: &v1beta1.AutoscalerSpec{
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
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:      true,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{Enabled: true},
					},
					Config: cfg,
				},
			},
			expectedWarnings: []string{
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - monitoring.coreos.com/servicemonitors: [*]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - monitoring.coreos.com/podmonitors: [*]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - nodes/metrics: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - services: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - endpoints: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - namespaces: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - networking.k8s.io/ingresses: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - nodes: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - pods: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - configmaps: [get]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - discovery.k8s.io/endpointslices: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:adm-warning-targetallocator - nonResourceURL: /metrics: [get]",
			},
		},
		{
			name:          "prom CR no admissions warning",
			shouldFailSar: false, // force SAR okay
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode:            v1beta1.ModeStatefulSet,
					UpgradeStrategy: "adhoc",
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []v1beta1.PortsSpec{
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
					Autoscaler: &v1beta1.AutoscalerSpec{
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
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:      true,
						PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{Enabled: true},
					},
					Config: cfg,
				},
			},
		},
		{
			name: "invalid mode with volume claim templates",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
						VolumeClaimTemplates: []v1.PersistentVolumeClaim{{}, {}},
					},
				},
			},
			expectedErr: "does not support the attribute 'volumeClaimTemplates'",
		},
		{
			name: "invalid mode with persistentVolumeClaimRetentionPolicy",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
						PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
							WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
							WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
						},
					},
				},
			},
			expectedErr: "does not support the attribute 'persistentVolumeClaimRetentionPolicy'",
		},
		{
			name: "invalid mode with tolerations",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Tolerations: []v1.Toleration{{}, {}},
					},
				},
			},
			expectedErr: "does not support the attribute 'tolerations'",
		},
		{
			name: "invalid mode with target allocator",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled: true,
					},
				},
			},
			expectedErr: "does not support the target allocation deployment",
		},
		{
			name: "invalid target allocator config",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeStatefulSet,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled: true,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Prometheus configuration is incorrect",
		},
		{
			name: "invalid target allocation strategy",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDaemonSet,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled:            true,
						AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
					},
				},
			},
			expectedErr: "mode is set to daemonset, which must be used with target allocation strategy per-node",
		},
		{
			name: "invalid port name",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &zero,
					},
				},
			},
			expectedErr: "maxReplicas should be defined and one or more",
		},
		{
			name: "invalid replicas, greater than max",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &five,
					},
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
					},
				},
			},
			expectedErr: "replicas must not be greater than maxReplicas",
		},
		{
			name: "invalid min replicas, greater than max",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						MinReplicas: &five,
					},
				},
			},
			expectedErr: "minReplicas must not be greater than maxReplicas",
		},
		{
			name: "invalid min replicas, lesser than 1",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						MinReplicas: &zero,
					},
				},
			},
			expectedErr: "minReplicas should be one or more",
		},
		{
			name: "invalid autoscaler scale down stablization window - <0",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleDown: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &minusOne,
							},
						},
					},
				},
			},
			expectedErr: "scaleDown.stabilizationWindowSeconds should be >=0 and <=3600",
		},
		{
			name: "invalid autoscaler scale down stablization window - >3600",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleDown: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &maxInt,
							},
						},
					},
				},
			},
			expectedErr: "scaleDown.stabilizationWindowSeconds should be >=0 and <=3600",
		},
		{
			name: "invalid autoscaler scale up stablization window - <0",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleUp: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &minusOne,
							},
						},
					},
				},
			},
			expectedErr: "scaleUp.stabilizationWindowSeconds should be >=0 and <=3600",
		},
		{
			name: "invalid autoscaler scale up stablization window - >3600",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
							ScaleUp: &autoscalingv2.HPAScalingRules{
								StabilizationWindowSeconds: &maxInt,
							},
						},
					},
				},
			},
			expectedErr: "scaleUp.stabilizationWindowSeconds should be >=0 and <=3600",
		},
		{
			name: "invalid autoscaler target cpu utilization",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas:          &three,
						TargetCPUUtilization: &zero,
					},
				},
			},
			expectedErr: "targetCPUUtilization should be greater than 0",
		},
		{
			name: "invalid autoscaler target memory utilization",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas:             &three,
						TargetMemoryUtilization: &zero,
					},
				},
			},
			expectedErr: "targetMemoryUtilization should be greater than 0",
		},
		{
			name: "autoscaler minReplicas is less than maxReplicas",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &one,
						MinReplicas: &five,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas must not be greater than maxReplicas",
		},
		{
			name: "invalid autoscaler metric type",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Metrics: []v1beta1.MetricSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Metrics: []v1beta1.MetricSpec{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Autoscaler: &v1beta1.AutoscalerSpec{
						MaxReplicas: &three,
						Metrics: []v1beta1.MetricSpec{
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
			name: "invalid deployment mode incompatible with ingress settings",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressTypeIngress,
					},
				},
			},
			expectedErr: fmt.Sprintf("Ingress can only be used in combination with the modes: %s, %s, %s", v1beta1.ModeDeployment, v1beta1.ModeDaemonSet, v1beta1.ModeStatefulSet),
		},
		{
			name: "invalid mode with priorityClassName",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						PriorityClassName: "test-class",
					},
				},
			},
			expectedErr: "does not support the attribute 'priorityClassName'",
		},
		{
			name: "invalid mode with affinity",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						InitialDelaySeconds: &minusOne,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe InitialDelaySeconds configuration is incorrect",
		},
		{
			name: "invalid InitialDelaySeconds readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						InitialDelaySeconds: &minusOne,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe InitialDelaySeconds configuration is incorrect",
		},
		{
			name: "invalid PeriodSeconds",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						PeriodSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe PeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid PeriodSeconds readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						PeriodSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe PeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid TimeoutSeconds",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						TimeoutSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TimeoutSeconds configuration is incorrect",
		},
		{
			name: "invalid TimeoutSeconds readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						TimeoutSeconds: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe TimeoutSeconds configuration is incorrect",
		},
		{
			name: "invalid SuccessThreshold",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						SuccessThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe SuccessThreshold configuration is incorrect",
		},
		{
			name: "invalid SuccessThreshold readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						SuccessThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe SuccessThreshold configuration is incorrect",
		},
		{
			name: "invalid FailureThreshold",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						FailureThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe FailureThreshold configuration is incorrect",
		},
		{
			name: "invalid FailureThreshold readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						FailureThreshold: &zero,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe FailureThreshold configuration is incorrect",
		},
		{
			name: "invalid TerminationGracePeriodSeconds",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					LivenessProbe: &v1beta1.Probe{
						TerminationGracePeriodSeconds: &zero64,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec LivenessProbe TerminationGracePeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid TerminationGracePeriodSeconds readiness",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					ReadinessProbe: &v1beta1.Probe{
						TerminationGracePeriodSeconds: &zero64,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec ReadinessProbe TerminationGracePeriodSeconds configuration is incorrect",
		},
		{
			name: "invalid AdditionalContainers",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Ingress: v1beta1.Ingress{
						RuleType: v1beta1.IngressRuleTypeSubdomain,
					},
				},
			},
			expectedErr: "a valid Ingress hostname has to be defined for subdomain ruleType",
		},
		{
			name: "invalid updateStrategy for Deployment mode",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeDeployment,
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
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeStatefulSet,
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
		{
			name: "missing port for ingress type",
			otelcol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
							{
								ServicePort: v1.ServicePort{},
							},
						},
					},
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressTypeIngress,
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
	}

	bv := func(_ context.Context, collector v1beta1.OpenTelemetryCollector) admission.Warnings {
		var warnings admission.Warnings
		cfg := config.New(
			config.WithCollectorImage("default-collector"),
			config.WithTargetAllocatorImage("default-ta-allocator"),
		)
		params := manifests.Params{
			Log:     logr.Discard(),
			Config:  cfg,
			OtelCol: collector,
		}
		_, err := collectorManifests.Build(params)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		return nil
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := v1beta1.NewCollectorWebhook(
				logr.Discard(),
				testScheme,
				config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
				getReviewer(test.shouldFailSar),
				nil,
				bv,
				nil,
			)
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

func TestOTELColValidateUpdateWebhook(t *testing.T) {
	tests := []struct { //nolint:govet
		name             string
		otelcolOld       v1beta1.OpenTelemetryCollector
		otelcolNew       v1beta1.OpenTelemetryCollector
		expectedErr      string
		expectedWarnings []string
		shouldFailSar    bool
	}{
		{
			name: "mode should not be changed",
			otelcolOld: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{Mode: v1beta1.ModeStatefulSet},
			},
			otelcolNew: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{Mode: v1beta1.ModeDeployment},
			},
			expectedErr: "which does not support modification",
		},
	}

	bv := func(_ context.Context, collector v1beta1.OpenTelemetryCollector) admission.Warnings {
		var warnings admission.Warnings
		cfg := config.New(
			config.WithCollectorImage("default-collector"),
			config.WithTargetAllocatorImage("default-ta-allocator"),
		)
		params := manifests.Params{
			Log:     logr.Discard(),
			Config:  cfg,
			OtelCol: collector,
		}
		_, err := collectorManifests.Build(params)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		return nil
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := v1beta1.NewCollectorWebhook(
				logr.Discard(),
				testScheme,
				config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
				getReviewer(test.shouldFailSar),
				nil,
				bv,
				nil,
			)
			ctx := context.Background()
			warnings, err := cvw.ValidateUpdate(ctx, &test.otelcolOld, &test.otelcolNew)
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
