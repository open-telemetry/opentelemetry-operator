// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var logger = logf.Log.WithName("unit-tests")

func TestUpgrade0_122_0(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *v1beta1.OpenTelemetryCollector
		expectedConfig *v1beta1.OpenTelemetryCollector
	}{
		{
			name: "should remove address field from metrics config",
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
										"address": "0.0.0.0:8888",
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
		},
		{
			name: "should not modify config when metrics address is empty",
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
										"address": "",
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
										"address": "",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should not modify config when prometheus reader is already configured",
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
		},
		{
			name: "should not modify config when OTLP is configured",
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
												"periodic": map[string]interface{}{
													"interval": "30000",
													"exporter": map[string]interface{}{
														"otlp": map[string]interface{}{
															"endpoint": "localhost:4317",
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
												"periodic": map[string]interface{}{
													"interval": "30000",
													"exporter": map[string]interface{}{
														"otlp": map[string]interface{}{
															"endpoint": "localhost:4317",
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
			name: "should not modify config when metrics config is missing",
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
								Object: map[string]interface{}{},
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
								Object: map[string]interface{}{},
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

			result, err := upgrade0_122_0(up, tt.initialConfig)
			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.initialConfig.Spec.Config.Service.Telemetry != nil &&
				tt.initialConfig.Spec.Config.Service.Telemetry.Object != nil {
				metrics, ok := tt.initialConfig.Spec.Config.Service.Telemetry.Object["metrics"].(map[string]interface{})
				if ok {
					address, ok := metrics["address"].(string)
					if ok && address != "" {
						assert.Equal(t, "", address, "address field should be set to empty string")
					}
				}
			}

			assert.Equal(t, tt.expectedConfig.Spec.Config.Service.Telemetry, result.Spec.Config.Service.Telemetry)
		})
	}
}
