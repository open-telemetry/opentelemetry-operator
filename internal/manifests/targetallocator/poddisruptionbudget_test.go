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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
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
		for _, strategy := range []v1beta1.TargetAllocatorAllocationStrategy{v1beta1.TargetAllocatorAllocationStrategyPerNode, v1beta1.TargetAllocatorAllocationStrategyConsistentHashing} {
			t.Run(fmt.Sprintf("%s-%s", strategy, test.name), func(t *testing.T) {
				targetAllocator := v1alpha1.TargetAllocator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-instance",
					},
					Spec: v1alpha1.TargetAllocatorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
								MinAvailable:   test.MinAvailable,
								MaxUnavailable: test.MaxUnavailable,
							},
						},
						AllocationStrategy: strategy,
					},
				}
				configuration := config.New()
				pdb, err := PodDisruptionBudget(manifests.Params{
					Log:             logger,
					Config:          configuration,
					TargetAllocator: targetAllocator,
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
}

func TestPDBWithNotValidStrategy(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			targetAllocator := v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-instance",
				},
				Spec: v1alpha1.TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable:   test.MinAvailable,
							MaxUnavailable: test.MaxUnavailable,
						},
					},
					AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
				},
			}
			configuration := config.New()
			pdb, err := PodDisruptionBudget(manifests.Params{
				Log:             logger,
				Config:          configuration,
				TargetAllocator: targetAllocator,
			})

			// verify
			assert.Error(t, err)
			assert.Nil(t, pdb)
		})
	}
}

func TestNoPDB(t *testing.T) {
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
		},
	}
	configuration := config.New()
	pdb, err := PodDisruptionBudget(manifests.Params{
		Log:             logger,
		Config:          configuration,
		TargetAllocator: targetAllocator,
	})

	// verify
	assert.NoError(t, err)
	assert.Nil(t, pdb)
}
