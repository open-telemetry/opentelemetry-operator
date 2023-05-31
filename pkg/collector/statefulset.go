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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// StatefulSet builds the statefulset for the given instance.
func StatefulSet(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) appsv1.StatefulSet {
	name := naming.Collector(otelcol)
	labels := Labels(otelcol, name, cfg.LabelsFilter())

	annotations := Annotations(otelcol)
	podAnnotations := PodAnnotations(otelcol)

	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: naming.Service(otelcol),
			Selector: &metav1.LabelSelector{
				MatchLabels: SelectorLabels(otelcol),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountName(otelcol),
					Containers:         []corev1.Container{Container(cfg, logger, otelcol, true)},
					Volumes:            Volumes(cfg, otelcol),
					DNSPolicy:          getDNSPolicy(otelcol),
					HostNetwork:        otelcol.Spec.HostNetwork,
					Tolerations:        otelcol.Spec.Tolerations,
					NodeSelector:       otelcol.Spec.NodeSelector,
					SecurityContext:    otelcol.Spec.PodSecurityContext,
					PriorityClassName:  otelcol.Spec.PriorityClassName,
					Affinity:           otelcol.Spec.Affinity,
				},
			},
			Replicas:             otelcol.Spec.Replicas,
			PodManagementPolicy:  "Parallel",
			VolumeClaimTemplates: VolumeClaimTemplates(otelcol),
		},
	}
}
