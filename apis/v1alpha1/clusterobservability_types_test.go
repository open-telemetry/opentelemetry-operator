// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func TestClusterObservabilityExporterJSONRoundTrip(t *testing.T) {
	exporterJSON := `{
		"endpoint": "https://otel.example.com:4318",
		"headers": {"Authorization": "Bearer token"},
		"sending_queue": {
			"enabled": true,
			"num_consumers": 2,
			"block_on_overflow": true
		}
	}`

	var co ClusterObservability
	if err := json.Unmarshal([]byte(`{"spec":{"exporter":`+exporterJSON+`}}`), &co); err != nil {
		t.Fatalf("unmarshal ClusterObservability: %v", err)
	}

	resource, err := json.Marshal(&co)
	if err != nil {
		t.Fatalf("marshal ClusterObservability: %v", err)
	}

	var roundTripped ClusterObservability
	if err := json.Unmarshal(resource, &roundTripped); err != nil {
		t.Fatalf("unmarshal round-tripped ClusterObservability: %v", err)
	}
	got := roundTripped.Spec.Exporter.Object
	want := map[string]any{
		"endpoint": "https://otel.example.com:4318",
		"headers":  map[string]any{"Authorization": "Bearer token"},
		"sending_queue": map[string]any{
			"enabled":           true,
			"num_consumers":     float64(2),
			"block_on_overflow": true,
		},
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("exporter mismatch: got %#v, want %#v", got, want)
	}
}

func TestClusterObservabilityExporterDeepCopy(t *testing.T) {
	original := &ClusterObservability{
		Spec: ClusterObservabilitySpec{
			Exporter: v1beta1.AnyConfig{Object: map[string]any{
				"sending_queue": map[string]any{"enabled": true},
			}},
		},
	}

	copied := original.DeepCopy()
	copied.Spec.Exporter.Object["sending_queue"].(map[string]any)["enabled"] = false

	if !original.Spec.Exporter.Object["sending_queue"].(map[string]any)["enabled"].(bool) {
		t.Fatal("deep copy mutated the original exporter")
	}
}
