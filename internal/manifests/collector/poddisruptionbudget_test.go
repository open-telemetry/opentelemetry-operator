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
	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestPDB(t *testing.T) {
	type test struct {
		name           string
		MinAvailable   *intstr.IntOrString
		MaxUnavailable *intstr.IntOrString
	}
	tests := []test{
		{
			name: "MinAvailable-int",
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
		{
			name: "MinAvailable-string",
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "10%",
			},
		},
		{
			name: "MaxUnavailable-int",
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
		{
			name: "MaxUnavailable-string",
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "10%",
			},
		},
	}

	otelcols := []v1alpha1.OpenTelemetryCollector{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		},
	}

	for _, otelcol := range otelcols {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				otelcol.Spec.PodDisruptionBudget = &v1alpha1.PodDisruptionBudgetSpec{
					MinAvailable:   test.MinAvailable,
					MaxUnavailable: test.MaxUnavailable,
				}
				configuration := config.New()
				raw := PodDisruptionBudget(manifests.Params{
					Log:     logger,
					Config:  configuration,
					OtelCol: otelcol,
				})

				pdb := raw.(*policyV1.PodDisruptionBudget)
				// verify
				assert.Equal(t, "my-instance-collector", pdb.Name)
				assert.Equal(t, "my-instance-collector", pdb.Labels["app.kubernetes.io/name"])
				assert.Equal(t, test.MinAvailable, pdb.Spec.MinAvailable)
				assert.Equal(t, test.MaxUnavailable, pdb.Spec.MaxUnavailable)
			})
		}
	}
}
