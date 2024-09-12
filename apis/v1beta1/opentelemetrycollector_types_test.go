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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpenTelemetryCollector_MarshalJSON(t *testing.T) {
	collector := &OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "opentelemetry.io/v1beta1",
			Kind:       "OpenTelemetryCollector",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-collector",
			Namespace: "default",
		},
		Spec: OpenTelemetryCollectorSpec{
			Config: Config{
				Receivers: AnyConfig{
					Object: map[string]interface{}{
						"otlp": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": nil,
							},
						},
					},
				},
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"batch": nil,
					},
				},
				Exporters: AnyConfig{
					Object: map[string]interface{}{
						"otlp": map[string]interface{}{
							"endpoint": "otelcol:4317",
						},
					},
				},
				Service: Service{
					Pipelines: map[string]*Pipeline{
						"traces": {
							Receivers:  []string{"otlp"},
							Processors: []string{"batch"},
							Exporters:  []string{"otlp"},
						},
					},
				},
			},
		},
	}
	marshaledJSON, err := json.Marshal(collector)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(marshaledJSON, &result)
	assert.NoError(t, err)

	spec, ok := result["spec"].(map[string]interface{})
	assert.True(t, ok, "spec should be a JSON object")

	config, ok := spec["config"].(map[string]interface{})
	assert.True(t, ok, "config should be a JSON object")

	receivers, ok := config["receivers"].(map[string]interface{})
	assert.True(t, ok, "receivers should be present in config")
	assert.Contains(t, receivers, "otlp")

	service, ok := config["service"].(map[string]interface{})
	assert.True(t, ok, "service should be present in config")
	assert.Contains(t, service, "pipelines")
}

func TestOpenTelemetryCollectorSpec_MarshalJSON(t *testing.T) {
	spec := &OpenTelemetryCollectorSpec{
		Mode: "deployment",
		Config: Config{
			Receivers: AnyConfig{
				Object: map[string]interface{}{
					"otlp": map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal OpenTelemetryCollectorSpec: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON data: %v", err)
	}

	if mode, ok := result["mode"].(string); !ok || mode != "deployment" {
		t.Errorf("Expected mode to be 'deployment', got %v", result["mode"])
	}

	config, ok := result["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected config to be a map[string]interface{}, got %T", result["config"])
	}

	receivers, ok := config["receivers"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected receivers to be a map[string]interface{}, got %T", config["receivers"])
	}

	otlp, ok := receivers["otlp"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected otlp to be a map[string]interface{}, got %T", receivers["otlp"])
	}

	protocols, ok := otlp["protocols"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected protocols to be a map[string]interface{}, got %T", otlp["protocols"])
	}

	_, ok = protocols["grpc"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected grpc to be a map[string]interface{}, got %T", protocols["grpc"])
	}
}
