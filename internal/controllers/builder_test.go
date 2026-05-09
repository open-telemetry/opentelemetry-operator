// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"bytes"
	"testing"

	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// renderObjects serializes a list of client.Object into a multi-document YAML
// string suitable for diffing against a golden file. TypeMeta is populated
// from testScheme when known, so the generated fixtures stay readable even
// though the builder itself does not set apiVersion/kind on returned objects.
// Types not registered in testScheme (e.g. cert-manager) are emitted without
// apiVersion/kind.
func renderObjects(t *testing.T, objs []client.Object) string {
	t.Helper()
	var buf bytes.Buffer
	for i, obj := range objs {
		if i > 0 {
			buf.WriteString("---\n")
		}
		if obj.GetObjectKind().GroupVersionKind().Empty() {
			if gvks, _, err := testScheme.ObjectKinds(obj); err == nil && len(gvks) > 0 {
				obj.GetObjectKind().SetGroupVersionKind(gvks[0])
			}
		}
		out, err := sigsyaml.Marshal(obj)
		require.NoError(t, err)
		buf.Write(out)
	}
	return buf.String()
}

func TestBuildCollector(t *testing.T) {
	goodConfigYaml := `receivers:
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

	one := int32(1)
	trueVal := true

	tests := []struct {
		name     string
		instance v1beta1.OpenTelemetryCollector
		wantFile string
		wantErr  bool
	}{
		{
			name: "base case",
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
					NetworkPolicy: v1beta1.NetworkPolicy{
						Enabled: &trueVal,
					},
				},
			},
			wantFile: "build_collector_base.yaml",
		},
		{
			name: "ingress",
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
			wantFile: "build_collector_ingress.yaml",
		},
		{
			name: "specified service account case",
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
			wantFile: "build_collector_service_account.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				CollectorImage:          "default-collector",
				TargetAllocatorImage:    "default-ta-allocator",
				CollectorConfigMapEntry: "collector.yaml",
			}
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.instance,
			}
			got, err := BuildCollector(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			golden.Assert(t, renderObjects(t, got), tt.wantFile)
		})
	}
}

func TestBuildAll_OpAMPBridge(t *testing.T) {
	one := int32(1)

	tests := []struct {
		name     string
		instance v1alpha1.OpAMPBridge
		wantFile string
		wantErr  bool
	}{
		{
			name: "base case",
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
			wantFile: "build_opamp_bridge_base.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				CollectorImage:                    "default-collector",
				TargetAllocatorImage:              "default-ta-allocator",
				OperatorOpAMPBridgeImage:          "default-opamp-bridge",
				CollectorConfigMapEntry:           "collector.yaml",
				OperatorOpAMPBridgeConfigMapEntry: "remoteconfiguration.yaml",
				TargetAllocatorConfigMapEntry:     "targetallocator.yaml",
			}
			reconciler := NewOpAMPBridgeReconciler(OpAMPBridgeReconcilerParams{
				Log:    logr.Discard(),
				Config: cfg,
			})
			params := reconciler.getParams(tt.instance)
			got, err := BuildOpAMPBridge(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			golden.Assert(t, renderObjects(t, got), tt.wantFile)
		})
	}
}

func TestBuildCollectorTargetAllocatorCR(t *testing.T) {
	goodConfigYaml := `
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

	one := int32(1)

	tests := []struct {
		name         string
		instance     v1beta1.OpenTelemetryCollector
		wantFile     string
		featuregates []*colfeaturegate.Gate
		wantErr      bool
	}{
		{
			name: "base case",
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
			wantFile: "build_collector_ta_cr_base.yaml",
		},
		{
			name: "enable metrics case",
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
			wantFile:     "build_collector_ta_cr_enable_metrics.yaml",
			featuregates: []*colfeaturegate.Gate{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				CollectorImage:                "default-collector",
				TargetAllocatorImage:          "default-ta-allocator",
				CollectorConfigMapEntry:       "collector.yaml",
				TargetAllocatorConfigMapEntry: "targetallocator.yaml",
			}
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.instance,
			}
			targetAllocator, err := collector.TargetAllocator(params)
			require.NoError(t, err)
			params.TargetAllocator = targetAllocator
			featuregates := []*colfeaturegate.Gate{}
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
			golden.Assert(t, renderObjects(t, got), tt.wantFile)
		})
	}
}

func TestBuildTargetAllocator(t *testing.T) {
	tests := []struct {
		name         string
		instance     v1alpha1.TargetAllocator
		collector    *v1beta1.OpenTelemetryCollector
		wantFile     string
		featuregates []*colfeaturegate.Gate
		wantErr      bool
		cfg          config.Config
	}{
		{
			name: "base case",
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
			wantFile: "build_target_allocator_base.yaml",
			cfg: config.Config{
				CollectorImage:                "default-collector",
				TargetAllocatorImage:          "default-ta-allocator",
				TargetAllocatorConfigMapEntry: "targetallocator.yaml",
				CollectorConfigMapEntry:       "collector.yaml",
			},
		},
		{
			name: "enable metrics case",
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
			wantFile: "build_target_allocator_enable_metrics.yaml",
			cfg: config.Config{
				CollectorImage:                "default-collector",
				TargetAllocatorImage:          "default-ta-allocator",
				TargetAllocatorConfigMapEntry: "targetallocator.yaml",
				CollectorConfigMapEntry:       "collector.yaml",
			},
		},
		{
			name: "collector present",
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
			wantFile: "build_target_allocator_collector_present.yaml",
			cfg: config.Config{
				CollectorImage:                "default-collector",
				TargetAllocatorImage:          "default-ta-allocator",
				TargetAllocatorConfigMapEntry: "targetallocator.yaml",
				CollectorConfigMapEntry:       "collector.yaml",
			},
		},
		{
			name: "mtls",
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
			wantFile: "build_target_allocator_mtls.yaml",
			cfg: config.Config{
				CertManagerAvailability:       certmanager.Available,
				CollectorImage:                "default-collector",
				TargetAllocatorImage:          "default-ta-allocator",
				TargetAllocatorConfigMapEntry: "targetallocator.yaml",
				CollectorConfigMapEntry:       "collector.yaml",
			},
			featuregates: []*colfeaturegate.Gate{featuregate.EnableTargetAllocatorMTLS},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := targetallocator.Params{
				Log:             logr.Discard(),
				Config:          tt.cfg,
				TargetAllocator: tt.instance,
				Collector:       tt.collector,
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
			golden.Assert(t, renderObjects(t, got), tt.wantFile)
		})
	}
}
