// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"gotest.tools/v3/golden"
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
// apiVersion/kind. The top-level "status" field is stripped — the builder
// must never set status, and zero-value status structs (Deployment, etc.)
// would otherwise pollute the golden files.
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
		raw, err := sigsyaml.Marshal(obj)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, sigsyaml.Unmarshal(raw, &m))
		delete(m, "status")
		out, err := sigsyaml.Marshal(m)
		require.NoError(t, err)
		buf.Write(out)
	}
	return buf.String()
}

// loadInput reads a YAML fixture from testdata/ and unmarshals it into T.
// Used to keep test inputs out of Go source where possible.
func loadInput[T any](t *testing.T, file string) T {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	require.NoError(t, err)
	var obj T
	require.NoError(t, sigsyaml.Unmarshal(data, &obj))
	return obj
}

func TestBuildCollector(t *testing.T) {
	tests := []struct {
		name      string
		inputFile string
		wantFile  string
		wantErr   bool
	}{
		{
			name:      "base case",
			inputFile: "build_collector_base.input.yaml",
			wantFile:  "build_collector_base.yaml",
		},
		{
			name:      "ingress",
			inputFile: "build_collector_ingress.input.yaml",
			wantFile:  "build_collector_ingress.yaml",
		},
		{
			name:      "specified service account case",
			inputFile: "build_collector_service_account.input.yaml",
			wantFile:  "build_collector_service_account.yaml",
		},
		{
			name:      "affinity",
			inputFile: "build_collector_affinity.input.yaml",
			wantFile:  "build_collector_affinity.yaml",
		},
		{
			name:      "node selector",
			inputFile: "build_collector_node_selector.input.yaml",
			wantFile:  "build_collector_node_selector.yaml",
		},
		{
			name:      "args",
			inputFile: "build_collector_args.input.yaml",
			wantFile:  "build_collector_args.yaml",
		},
		{
			name:      "init containers",
			inputFile: "build_collector_init_containers.input.yaml",
			wantFile:  "build_collector_init_containers.yaml",
		},
		{
			name:      "pod DNS config",
			inputFile: "build_collector_dns_config.input.yaml",
			wantFile:  "build_collector_dns_config.yaml",
		},
		{
			name:      "pod annotations",
			inputFile: "build_collector_pod_annotations.input.yaml",
			wantFile:  "build_collector_pod_annotations.yaml",
		},
		{
			name:      "additional containers",
			inputFile: "build_collector_additional_containers.input.yaml",
			wantFile:  "build_collector_additional_containers.yaml",
		},
		{
			name:      "daemonset features",
			inputFile: "build_collector_daemonset_features.input.yaml",
			wantFile:  "build_collector_daemonset_features.yaml",
		},
		{
			name:      "extension",
			inputFile: "build_collector_extension.input.yaml",
			wantFile:  "build_collector_extension.yaml",
		},
		{
			name:      "http route",
			inputFile: "build_collector_http_route.input.yaml",
			wantFile:  "build_collector_http_route.yaml",
		},
		{
			name:      "configmaps",
			inputFile: "build_collector_configmaps.input.yaml",
			wantFile:  "build_collector_configmaps.yaml",
		},
		{
			name:      "ip families",
			inputFile: "build_collector_ip_families.input.yaml",
			wantFile:  "build_collector_ip_families.yaml",
		},
		{
			name:      "image with version tag",
			inputFile: "build_collector_image_version.input.yaml",
			wantFile:  "build_collector_image_version.yaml",
		},
		{
			name:      "statefulset",
			inputFile: "build_collector_statefulset.input.yaml",
			wantFile:  "build_collector_statefulset.yaml",
		},
		{
			name:      "statefulset features",
			inputFile: "build_collector_statefulset_features.input.yaml",
			wantFile:  "build_collector_statefulset_features.yaml",
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
				OtelCol: loadInput[v1beta1.OpenTelemetryCollector](t, tt.inputFile),
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
	tests := []struct {
		name      string
		inputFile string
		wantFile  string
		wantErr   bool
	}{
		{
			name:      "base case",
			inputFile: "build_opamp_bridge_base.input.yaml",
			wantFile:  "build_opamp_bridge_base.yaml",
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
			params := reconciler.getParams(loadInput[v1alpha1.OpAMPBridge](t, tt.inputFile))
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
	tests := []struct {
		name         string
		inputFile    string
		wantFile     string
		featuregates []*colfeaturegate.Gate
		wantErr      bool
	}{
		{
			name:      "base case",
			inputFile: "build_collector_ta_cr_base.input.yaml",
			wantFile:  "build_collector_ta_cr_base.yaml",
		},
		{
			name:         "enable metrics case",
			inputFile:    "build_collector_ta_cr_enable_metrics.input.yaml",
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
				OtelCol: loadInput[v1beta1.OpenTelemetryCollector](t, tt.inputFile),
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
			golden.Assert(t, renderObjects(t, got), tt.wantFile)
		})
	}
}

func TestBuildTargetAllocator(t *testing.T) {
	defaultCfg := config.Config{
		CollectorImage:                "default-collector",
		TargetAllocatorImage:          "default-ta-allocator",
		TargetAllocatorConfigMapEntry: "targetallocator.yaml",
		CollectorConfigMapEntry:       "collector.yaml",
	}

	tests := []struct {
		name          string
		inputFile     string
		collectorFile string
		wantFile      string
		featuregates  []*colfeaturegate.Gate
		wantErr       bool
		cfg           config.Config
	}{
		{
			name:      "base case",
			inputFile: "build_target_allocator_base.input.yaml",
			wantFile:  "build_target_allocator_base.yaml",
			cfg:       defaultCfg,
		},
		{
			name:      "enable metrics case",
			inputFile: "build_target_allocator_enable_metrics.input.yaml",
			wantFile:  "build_target_allocator_enable_metrics.yaml",
			cfg:       defaultCfg,
		},
		{
			name:          "collector present",
			inputFile:     "build_target_allocator_minimal.input.yaml",
			collectorFile: "build_target_allocator_collector.input.yaml",
			wantFile:      "build_target_allocator_collector_present.yaml",
			cfg:           defaultCfg,
		},
		{
			name:      "rich features",
			inputFile: "build_target_allocator_features.input.yaml",
			wantFile:  "build_target_allocator_features.yaml",
			cfg:       defaultCfg,
		},
		{
			name:          "mtls",
			inputFile:     "build_target_allocator_minimal.input.yaml",
			collectorFile: "build_target_allocator_collector.input.yaml",
			wantFile:      "build_target_allocator_mtls.yaml",
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
			var collector *v1beta1.OpenTelemetryCollector
			if tt.collectorFile != "" {
				c := loadInput[v1beta1.OpenTelemetryCollector](t, tt.collectorFile)
				collector = &c
			}
			params := targetallocator.Params{
				Log:             logr.Discard(),
				Config:          tt.cfg,
				TargetAllocator: loadInput[v1alpha1.TargetAllocator](t, tt.inputFile),
				Collector:       collector,
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
