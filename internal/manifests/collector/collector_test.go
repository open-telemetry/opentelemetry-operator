// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelColFeatureGate "go.opentelemetry.io/collector/featuregate"
	v1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRbac "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	irbac "github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestNeedsCheckSaPermissions(t *testing.T) {
	tests := []struct {
		name     string
		params   manifests.Params
		expected bool
	}{
		{
			name: "should return true when all conditions are met",
			params: manifests.Params{
				ErrorAsWarning: true,
				Config:         config.New(config.WithRBACPermissions(autoRbac.NotAvailable)),
				Reviewer:       &mockReviewer{},
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "test-sa",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "should return false when ErrorAsWarning is false",
			params: manifests.Params{
				ErrorAsWarning: false,
				Config:         config.New(config.WithRBACPermissions(autoRbac.NotAvailable)),
				Reviewer:       &mockReviewer{},
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "test-sa",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "should return false when RBAC is available",
			params: manifests.Params{
				ErrorAsWarning: true,
				Config:         config.New(config.WithRBACPermissions(autoRbac.Available)),
				Reviewer:       &mockReviewer{},
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "test-sa",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "should return false when Reviewer is nil",
			params: manifests.Params{
				ErrorAsWarning: true,
				Config:         config.New(config.WithRBACPermissions(autoRbac.NotAvailable)),
				Reviewer:       nil,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "test-sa",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "should return false when ServiceAccount is empty",
			params: manifests.Params{
				ErrorAsWarning: true,
				Config:         config.New(config.WithRBACPermissions(autoRbac.NotAvailable)),
				Reviewer:       &mockReviewer{},
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "",
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsCheckSaPermissions(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type mockReviewer struct{}

var _ irbac.SAReviewer = &mockReviewer{}

func (m *mockReviewer) CheckPolicyRules(ctx context.Context, serviceAccount, serviceAccountNamespace string, rules ...*rbacv1.PolicyRule) ([]*v1.SubjectAccessReview, error) {
	return nil, fmt.Errorf("error checking policy rules")
}

func (m *mockReviewer) CanAccess(ctx context.Context, serviceAccount, serviceAccountNamespace string, res *v1.ResourceAttributes, nonResourceAttributes *v1.NonResourceAttributes) (*v1.SubjectAccessReview, error) {
	return nil, nil
}

func TestBuild(t *testing.T) {
	logger := logr.Discard()
	tests := []struct {
		name            string
		params          manifests.Params
		expectedObjects int
		wantErr         bool
		featureGate     *otelColFeatureGate.Gate
	}{
		{
			name: "deployment mode builds expected manifests",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeDeployment,
					},
				},
				Config: config.New(),
			},
			expectedObjects: 5,
			wantErr:         false,
		},
		{
			name: "statefulset mode builds expected manifests",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeStatefulSet,
					},
				},
				Config: config.New(),
			},
			expectedObjects: 5,
			wantErr:         false,
		},
		{
			name: "sidecar mode skips deployment manifests",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeSidecar,
					},
				},
				Config: config.New(),
			},
			expectedObjects: 3,
			wantErr:         false,
		},
		{
			name: "rbac available adds cluster role manifests",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeDeployment,
						Config: v1beta1.Config{
							Processors: &v1beta1.AnyConfig{
								Object: map[string]any{
									"k8sattributes": map[string]any{},
								},
							},
							Service: v1beta1.Service{
								Pipelines: map[string]*v1beta1.Pipeline{
									"traces": {
										Processors: []string{"k8sattributes"},
									},
								},
							},
						},
					},
				},
				Config: config.New(config.WithRBACPermissions(autoRbac.Available)),
			},
			expectedObjects: 7,
			wantErr:         false,
		},
		{
			name: "metrics enabled adds monitoring service monitor",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeDeployment,
						Observability: v1beta1.ObservabilitySpec{
							Metrics: v1beta1.MetricsConfigSpec{
								EnableMetrics: true,
							},
						},
					},
				},
				Config: config.New(config.WithPrometheusCRAvailability(prometheus.Available)),
			},
			expectedObjects: 6,
			wantErr:         false,
			featureGate:     featuregate.PrometheusOperatorIsAvailable,
		},
		{
			name: "metrics enabled adds service monitors",
			params: manifests.Params{
				Log: logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode: v1beta1.ModeDeployment,
						Observability: v1beta1.ObservabilitySpec{
							Metrics: v1beta1.MetricsConfigSpec{
								EnableMetrics: true,
							},
						},
						Config: v1beta1.Config{
							Exporters: v1beta1.AnyConfig{
								Object: map[string]any{
									"prometheus": map[string]any{
										"endpoint": "1.2.3.4:1234",
									},
								},
							},
							Service: v1beta1.Service{
								Pipelines: map[string]*v1beta1.Pipeline{
									"metrics": {
										Exporters: []string{"prometheus"},
									},
								},
							},
						},
					},
				},
				Config: config.New(config.WithPrometheusCRAvailability(prometheus.Available)),
			},
			expectedObjects: 9,
			wantErr:         false,
			featureGate:     featuregate.PrometheusOperatorIsAvailable,
		},
		{
			name: "check sa permissions",
			params: manifests.Params{
				ErrorAsWarning: true,
				Reviewer:       &mockReviewer{},
				Log:            logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ServiceAccount: "test-sa",
						},
						Mode: v1beta1.ModeDeployment,
						Observability: v1beta1.ObservabilitySpec{
							Metrics: v1beta1.MetricsConfigSpec{
								EnableMetrics: true,
							},
						},
						Config: v1beta1.Config{
							Processors: &v1beta1.AnyConfig{
								Object: map[string]any{
									"k8sattributes": map[string]any{},
								},
							},
							Service: v1beta1.Service{
								Pipelines: map[string]*v1beta1.Pipeline{
									"metrics": {
										Processors: []string{"k8sattributes"},
									},
								},
							},
						},
					},
				},
				Config: config.New(config.WithPrometheusCRAvailability(prometheus.Available)),
			},
			expectedObjects: 9,
			wantErr:         true,
			featureGate:     featuregate.PrometheusOperatorIsAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.featureGate != nil {
				err := otelColFeatureGate.GlobalRegistry().Set(tt.featureGate.ID(), true)
				require.NoError(t, err)
				defer func() {
					err := otelColFeatureGate.GlobalRegistry().Set(tt.featureGate.ID(), false)
					require.NoError(t, err)
				}()
			}

			objects, err := Build(tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, objects, tt.expectedObjects)
		})
	}
}
