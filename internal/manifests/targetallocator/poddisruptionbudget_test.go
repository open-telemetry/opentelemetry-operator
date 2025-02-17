// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
)

type expected struct {
	MaxUnavailable *intstr.IntOrString
	MinAvailable   *intstr.IntOrString
}
type test struct {
	name     string
	spec     *v1beta1.PodDisruptionBudgetSpec
	expected expected
}

var tests = []test{
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
							PodDisruptionBudget: test.spec.DeepCopy(),
						},
						AllocationStrategy: strategy,
					},
				}
				configuration := config.New()
				pdb, err := PodDisruptionBudget(Params{
					Log:             logger,
					Config:          configuration,
					TargetAllocator: targetAllocator,
				})

				// verify
				assert.NoError(t, err)
				assert.Equal(t, "my-instance-targetallocator", pdb.Name)
				assert.Equal(t, "my-instance-targetallocator", pdb.Labels["app.kubernetes.io/name"])
				assert.Equal(t, test.expected.MinAvailable, pdb.Spec.MinAvailable)
				assert.Equal(t, test.expected.MaxUnavailable, pdb.Spec.MaxUnavailable)
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
						PodDisruptionBudget: test.spec.DeepCopy(),
					},
					AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
				},
			}
			configuration := config.New()
			pdb, err := PodDisruptionBudget(Params{
				Log:             logger,
				Config:          configuration,
				TargetAllocator: targetAllocator,
			})

			// verify that we error if the spec is set here
			if test.spec.DeepCopy() != nil {
				assert.Error(t, err)
			} else {
				// Should be no error if no one is attempting to set a PDB here
				assert.NoError(t, err)
			}
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
	pdb, err := PodDisruptionBudget(Params{
		Log:             logger,
		Config:          configuration,
		TargetAllocator: targetAllocator,
	})

	// verify
	assert.NoError(t, err)
	assert.Nil(t, pdb)
}
