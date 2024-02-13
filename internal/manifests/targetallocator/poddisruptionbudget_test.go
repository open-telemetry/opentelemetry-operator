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

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

type test struct {
	name           string
	MinAvailable   *intstr.IntOrString
	MaxUnavailable *intstr.IntOrString
}

var tests = []test{
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

func TestPDBWithValidStrategy(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			otelcol := v1alpha2.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-instance",
				},
				Spec: v1alpha2.OpenTelemetryCollectorSpec{
					TargetAllocator: v1alpha2.TargetAllocatorEmbedded{
						PodDisruptionBudget: &v1alpha2.PodDisruptionBudgetSpec{
							MinAvailable:   test.MinAvailable,
							MaxUnavailable: test.MaxUnavailable,
						},
						AllocationStrategy: v1alpha2.TargetAllocatorAllocationStrategyConsistentHashing,
					},
				},
			}
			configuration := config.New()
			pdb, err := PodDisruptionBudget(manifests.Params{
				Log:     logger,
				Config:  configuration,
				OtelCol: otelcol,
			})

			// verify
			assert.NoError(t, err)
			assert.Equal(t, "my-instance-targetallocator", pdb.Name)
			assert.Equal(t, "my-instance-targetallocator", pdb.Labels["app.kubernetes.io/name"])
			assert.Equal(t, test.MinAvailable, pdb.Spec.MinAvailable)
			assert.Equal(t, test.MaxUnavailable, pdb.Spec.MaxUnavailable)
		})
	}
}

func TestPDBWithNotValidStrategy(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			otelcol := v1alpha2.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-instance",
				},
				Spec: v1alpha2.OpenTelemetryCollectorSpec{
					TargetAllocator: v1alpha2.TargetAllocatorEmbedded{
						PodDisruptionBudget: &v1alpha2.PodDisruptionBudgetSpec{
							MinAvailable:   test.MinAvailable,
							MaxUnavailable: test.MaxUnavailable,
						},
						AllocationStrategy: v1alpha2.TargetAllocatorAllocationStrategyLeastWeighted,
					},
				},
			}
			configuration := config.New()
			pdb, err := PodDisruptionBudget(manifests.Params{
				Log:     logger,
				Config:  configuration,
				OtelCol: otelcol,
			})

			// verify
			assert.Error(t, err)
			assert.Nil(t, pdb)
		})
	}
}

func TestNoPDB(t *testing.T) {
	otelcol := v1alpha2.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha2.OpenTelemetryCollectorSpec{
			TargetAllocator: v1alpha2.TargetAllocatorEmbedded{
				AllocationStrategy: v1alpha2.TargetAllocatorAllocationStrategyLeastWeighted,
			},
		},
	}
	configuration := config.New()
	pdb, err := PodDisruptionBudget(manifests.Params{
		Log:     logger,
		Config:  configuration,
		OtelCol: otelcol,
	})

	// verify
	assert.NoError(t, err)
	assert.Nil(t, pdb)
}
