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

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestPDB(t *testing.T) {
	type expected struct {
		MaxUnavailable *intstr.IntOrString
		MinAvailable   *intstr.IntOrString
	}
	type test struct {
		name     string
		spec     *v1beta1.PodDisruptionBudgetSpec
		expected expected
	}
	tests := []test{
		{
			name: "defaults",
			spec: nil,
			expected: expected{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			},
		},
		{
			name: "MinAvailable-int",
			expected: expected{
				MinAvailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			},
			spec: &v1beta1.PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			},
		},
		{
			name: "MinAvailable-string",
			expected: expected{
				MinAvailable: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "10%",
				},
			},
			spec: &v1beta1.PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "10%",
				},
			},
		},
		{
			name: "MaxUnavailable-int",
			expected: expected{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			},
			spec: &v1beta1.PodDisruptionBudgetSpec{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			},
		},
		{
			name: "MaxUnavailable-string",
			expected: expected{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "10%",
				},
			},
			spec: &v1beta1.PodDisruptionBudgetSpec{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "10%",
				},
			},
		},
	}

	otelcols := []v1beta1.OpenTelemetryCollector{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		},
	}

	for _, otelcol := range otelcols {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				otelcol.Spec.PodDisruptionBudget = test.spec.DeepCopy()
				configuration := config.New()
				pdb, err := PodDisruptionBudget(manifests.Params{
					Log:     logger,
					Config:  configuration,
					OtelCol: otelcol,
				})
				require.NoError(t, err)

				// verify
				assert.Equal(t, "my-instance-collector", pdb.Name)
				assert.Equal(t, "my-instance-collector", pdb.Labels["app.kubernetes.io/name"])
				assert.Equal(t, test.expected.MinAvailable, pdb.Spec.MinAvailable)
				assert.Equal(t, test.expected.MaxUnavailable, pdb.Spec.MaxUnavailable)
			})
		}
	}
}
