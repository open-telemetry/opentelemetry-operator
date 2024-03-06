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

package collector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
)

func TestTargetAllocator(t *testing.T) {
	objectMetadata := metav1.ObjectMeta{
		Name:      "name",
		Namespace: "namespace",
		Annotations: map[string]string{
			"annotation_key": "annotation_value",
		},
		Labels: map[string]string{
			"label_key": "label_value",
		},
	}
	otelcolConfig := v1beta1.Config{
		Receivers: v1beta1.AnyConfig{
			Object: map[string]interface{}{
				"prometheus": map[string]any{
					"config": map[string]any{
						"scrape_configs": []any{},
					},
				},
			},
		},
	}

	testCases := []struct {
		name    string
		input   v1beta1.OpenTelemetryCollector
		want    *v1beta1.TargetAllocator
		wantErr error
	}{
		{
			name: "disabled",
			input: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled: false,
					},
				},
			},
			want: nil,
		},
		{
			name: "metadata",
			input: v1beta1.OpenTelemetryCollector{
				ObjectMeta: objectMetadata,
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: otelcolConfig,
					TargetAllocator: v1beta1.TargetAllocatorEmbedded{
						Enabled: true,
					},
				},
			},
			want: &v1beta1.TargetAllocator{
				ObjectMeta: objectMetadata,
				Spec: v1beta1.TargetAllocatorSpec{
					CollectorSelector: metav1.LabelSelector{
						MatchLabels: manifestutils.SelectorLabels(objectMetadata, ComponentOpenTelemetryCollector),
					},
					ScrapeConfigs: []v1beta1.AnyConfig{},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			params := manifests.Params{
				OtelCol: testCase.input,
			}
			actual, err := TargetAllocator(params)
			assert.Equal(t, testCase.wantErr, err)
			assert.Equal(t, testCase.want, actual)
		})
	}
}

func TestGetScrapeConfigs(t *testing.T) {
	testCases := []struct {
		name    string
		input   v1beta1.Config
		want    []v1beta1.AnyConfig
		wantErr error
	}{
		{
			name: "empty scrape configs list",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{
								"scrape_configs": []any{},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{},
		},
		{
			name: "no scrape configs key",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{},
						},
					},
				},
			},
			wantErr: fmt.Errorf("no scrape_configs available as part of the configuration"),
		},
		{
			name: "one scrape config",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{
								"scrape_configs": []any{
									map[string]any{
										"job": "somejob",
									},
								},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{
				{Object: map[string]interface{}{"job": "somejob"}},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			configStr, err := testCase.input.Yaml()
			require.NoError(t, err)
			actual, err := getScrapeConfigs(configStr)
			assert.Equal(t, testCase.wantErr, err)
			assert.Equal(t, testCase.want, actual)
		})
	}
}
