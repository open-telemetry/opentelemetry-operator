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

	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func PodDisruptionBudget(params manifests.Params) (*policyV1.PodDisruptionBudget, error) {
	// defaulting webhook should set this if the strategy is compatible, but if unset then return nil.
	if params.TargetAllocator.Spec.PodDisruptionBudget == nil {
		params.Log.Info("pdb field is unset in Spec, skipping podDisruptionBudget creation")
		return nil, nil
	}

	// defaulter doesn't set PodDisruptionBudget if the strategy isn't valid,
	// if PodDisruptionBudget != nil and stategy isn't correct, users have set
	// it wrongly
	if params.TargetAllocator.Spec.AllocationStrategy != v1beta1.TargetAllocatorAllocationStrategyConsistentHashing &&
		params.TargetAllocator.Spec.AllocationStrategy != v1beta1.TargetAllocatorAllocationStrategyPerNode {
		params.Log.V(4).Info("current allocation strategy not compatible, skipping podDisruptionBudget creation")
		return nil, fmt.Errorf("target allocator pdb has been configured but the allocation strategy isn't not compatible")
	}

	name := naming.TAPodDisruptionBudget(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)
	configMap, err := ConfigMap(params)
	if err != nil {
		params.Log.Info("failed to construct target allocator config map for annotations")
		configMap = nil
	}
	annotations := Annotations(params.TargetAllocator, configMap, params.Config.AnnotationsFilter())

	objectMeta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   params.TargetAllocator.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return &policyV1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyV1.PodDisruptionBudgetSpec{
			MinAvailable:   params.TargetAllocator.Spec.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: params.TargetAllocator.Spec.PodDisruptionBudget.MaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.TASelectorLabels(params.TargetAllocator, ComponentOpenTelemetryTargetAllocator),
			},
		},
	}, nil
}
