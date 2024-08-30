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

package collector

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

// TargetAllocator builds the TargetAllocator CR for the given instance.
func TargetAllocator(params manifests.Params) (*v1alpha1.TargetAllocator, error) {

	taSpec := params.OtelCol.Spec.TargetAllocator
	if !taSpec.Enabled {
		return nil, nil
	}

	return &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.OtelCol.Name,
			Namespace:   params.OtelCol.Namespace,
			Annotations: params.OtelCol.Annotations,
			Labels:      params.OtelCol.Labels,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Replicas:                  taSpec.Replicas,
				NodeSelector:              taSpec.NodeSelector,
				Resources:                 taSpec.Resources,
				ServiceAccount:            taSpec.ServiceAccount,
				Image:                     taSpec.Image,
				Affinity:                  taSpec.Affinity,
				SecurityContext:           taSpec.SecurityContext,
				PodSecurityContext:        taSpec.PodSecurityContext,
				TopologySpreadConstraints: taSpec.TopologySpreadConstraints,
				Tolerations:               taSpec.Tolerations,
				Env:                       taSpec.Env,
				PodAnnotations:            params.OtelCol.Spec.PodAnnotations,
				PodDisruptionBudget:       taSpec.PodDisruptionBudget,
			},
			AllocationStrategy: taSpec.AllocationStrategy,
			FilterStrategy:     taSpec.FilterStrategy,
			PrometheusCR:       taSpec.PrometheusCR,
			Observability:      taSpec.Observability,
		},
	}, nil
}
