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
	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func PodDisruptionBudget(params manifests.Params) client.Object {
	// defaulting webhook should always set this, but if unset then return nil.
	if params.OtelCol.Spec.PodDisruptionBudget == nil {
		params.Log.Info("pdb field is unset in Spec, skipping podDisruptionBudget creation")
		return nil
	}

	name := naming.Collector(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	annotations := Annotations(params.OtelCol)

	objectMeta := metav1.ObjectMeta{
		Name:        naming.PodDisruptionBudget(params.OtelCol.Name),
		Namespace:   params.OtelCol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return &policyV1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyV1.PodDisruptionBudgetSpec{
			MinAvailable:   params.OtelCol.Spec.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: params.OtelCol.Spec.PodDisruptionBudget.MaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: objectMeta.Labels,
			},
		},
	}
}
