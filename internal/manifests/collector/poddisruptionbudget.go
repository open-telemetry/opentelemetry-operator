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

	"github.com/open-telemetry/opentelemetry-operator/internal/api/convert"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func PodDisruptionBudget(params manifests.Params) (*policyV1.PodDisruptionBudget, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}
	// defaulting webhook should always set this, but if unset then return nil.
	if otelCol.Spec.PodDisruptionBudget == nil {
		params.Log.Info("pdb field is unset in Spec, skipping podDisruptionBudget creation")
		return nil, nil
	}

	name := naming.Collector(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, name, otelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	annotations, err := Annotations(otelCol)
	if err != nil {
		return nil, err
	}

	objectMeta := metav1.ObjectMeta{
		Name:        naming.PodDisruptionBudget(otelCol.Name),
		Namespace:   otelCol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return &policyV1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyV1.PodDisruptionBudgetSpec{
			MinAvailable:   otelCol.Spec.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: otelCol.Spec.PodDisruptionBudget.MaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: objectMeta.Labels,
			},
		},
	}, nil
}
