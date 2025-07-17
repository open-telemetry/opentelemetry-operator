// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func TestUpgrade0_130_0(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *v1beta1.OpenTelemetryCollector
		expectedConfig *v1beta1.OpenTelemetryCollector
	}{
		{
			name: "should set without_units if not set",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{
											map[string]interface{}{
												"pull": map[string]interface{}{
													"exporter": map[string]interface{}{
														"prometheus": map[string]interface{}{
															"host": "0.0.0.0",
															"port": int32(8888),
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
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{
											map[string]interface{}{
												"pull": map[string]interface{}{
													"exporter": map[string]interface{}{
														"prometheus": map[string]interface{}{
															"host":          "0.0.0.0",
															"port":          int32(8888),
															"without_units": true,
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
			},
		},
		{
			name: "should not set without_units if false",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{
											map[string]interface{}{
												"pull": map[string]interface{}{
													"exporter": map[string]interface{}{
														"prometheus": map[string]interface{}{
															"host":          "0.0.0.0",
															"port":          int32(8888),
															"without_units": false,
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
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{
											map[string]interface{}{
												"pull": map[string]interface{}{
													"exporter": map[string]interface{}{
														"prometheus": map[string]interface{}{
															"host":          "0.0.0.0",
															"port":          int32(8888),
															"without_units": false,
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
			},
		},
		{
			name: "should not set without_units if no prometheus exporter",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{},
									},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Service: v1beta1.Service{
							Telemetry: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"metrics": map[string]interface{}{
										"readers": []interface{}{},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := VersionUpgrade{
				Log: logger,
			}

			result, err := upgrade0_130_0(up, tt.initialConfig)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedConfig.Spec.Config.Service.Telemetry, result.Spec.Config.Service.Telemetry)
		})
	}
}
