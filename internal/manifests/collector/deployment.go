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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(params manifests.Params) *appsv1.Deployment {
	name := naming.Collector(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Common.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	annotations := Annotations(params.OtelCol)
	podAnnotations := PodAnnotations(params.OtelCol)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: params.OtelCol.Spec.Common.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ServiceAccountName(params.OtelCol),
					InitContainers:                params.OtelCol.Spec.Common.InitContainers,
					Containers:                    append(params.OtelCol.Spec.AdditionalContainers, Container(params.Config, params.Log, params.OtelCol, true)),
					Volumes:                       Volumes(params.Config, params.OtelCol),
					DNSPolicy:                     getDNSPolicy(params.OtelCol),
					HostNetwork:                   params.OtelCol.Spec.Common.HostNetwork,
					Tolerations:                   params.OtelCol.Spec.Common.Tolerations,
					NodeSelector:                  params.OtelCol.Spec.Common.NodeSelector,
					SecurityContext:               params.OtelCol.Spec.Common.PodSecurityContext,
					PriorityClassName:             params.OtelCol.Spec.Common.PriorityClassName,
					Affinity:                      params.OtelCol.Spec.Common.Affinity,
					TerminationGracePeriodSeconds: params.OtelCol.Spec.Common.TerminationGracePeriodSeconds,
					TopologySpreadConstraints:     params.OtelCol.Spec.Common.TopologySpreadConstraints,
				},
			},
		},
	}
}
