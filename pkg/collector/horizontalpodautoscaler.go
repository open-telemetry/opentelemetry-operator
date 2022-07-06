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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

const defaultCPUTarget int32 = 90

func HorizontalPodAutoscaler(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) autoscalingv1.HorizontalPodAutoscaler {
	labels := Labels(otelcol, cfg.LabelsFilter())
	labels["app.kubernetes.io/name"] = naming.Collector(otelcol)

	annotations := Annotations(otelcol)
	cpuTarget := defaultCPUTarget

	return autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.HorizontalPodAutoscaler(otelcol),
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "OpenTelemetryCollector",
				Name:       naming.OpenTelemetryCollector(otelcol),
			},

			MinReplicas:                    otelcol.Spec.Replicas,
			MaxReplicas:                    *otelcol.Spec.MaxReplicas,
			TargetCPUUtilizationPercentage: &cpuTarget,
		},
	}
}
