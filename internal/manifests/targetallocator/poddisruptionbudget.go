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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"

	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PodDisruptionBudget(params manifests.Params) (*policyV1.PodDisruptionBudget, error) {
	// defaulting webhook should set this if the strategy is compatible, but if unset then return nil.
	if params.OtelCol.Spec.TargetAllocator.PodDisruptionBudget == nil {
		params.Log.Info("pdb field is unset in Spec, skipping podDisruptionBudget creation")
		return nil, nil
	}

	// defaulter doesn't set PodDisruptionBudget if the strategy isn't valid,
	// if PodDisruptionBudget != nil and stategy isn't correct, users have set
	// it wrongly
	if params.OtelCol.Spec.TargetAllocator.AllocationStrategy != v1alpha2.TargetAllocatorAllocationStrategyConsistentHashing {
		params.Log.V(4).Info("current allocation strategy not compatible, skipping podDisruptionBudget creation")
		return nil, fmt.Errorf("target allocator pdb has been configured but the allocation strategy isn't not compatible")
	}

	name := naming.TAPodDisruptionBudget(params.OtelCol.Name)
	labels := Labels(params.OtelCol, name)

	annotations := Annotations(params.OtelCol, nil)

	objectMeta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   params.OtelCol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return &policyV1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyV1.PodDisruptionBudgetSpec{
			MinAvailable:   params.OtelCol.Spec.TargetAllocator.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: params.OtelCol.Spec.TargetAllocator.PodDisruptionBudget.MaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: SelectorLabels(params.OtelCol),
			},
		},
	}, nil
}
