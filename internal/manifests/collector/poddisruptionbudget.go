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
	"github.com/go-logr/logr"
	policyV1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func PodDisruptionBudget(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) client.Object {
	// defaulting webhook should always set this, but if unset then return nil.
	if otelcol.Spec.PodDisruptionBudget == nil {
		logger.Info("pdb field is unset in Spec, skipping podDisruptionBudget creation")
		return nil
	}

	name := naming.Collector(otelcol.Name)
	labels := Labels(otelcol, name, cfg.LabelsFilter())
	annotations := Annotations(otelcol)

	objectMeta := metav1.ObjectMeta{
		Name:        naming.PodDisruptionBudget(otelcol.Name),
		Namespace:   otelcol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	return &policyV1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyV1.PodDisruptionBudgetSpec{
			MinAvailable:   otelcol.Spec.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: otelcol.Spec.PodDisruptionBudget.MaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: objectMeta.Labels,
			},
		},
	}
}
