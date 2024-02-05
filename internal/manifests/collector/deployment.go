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

	"github.com/open-telemetry/opentelemetry-operator/internal/api/convert"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(params manifests.Params) (*appsv1.Deployment, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}
	name := naming.Collector(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, name, otelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	annotations, err := Annotations(otelCol)
	if err != nil {
		return nil, err
	}

	podAnnotations, err := PodAnnotations(otelCol)
	if err != nil {
		return nil, err
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: otelCol.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(otelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			},
			Strategy: otelCol.Spec.DeploymentUpdateStrategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ServiceAccountName(otelCol),
					InitContainers:                otelCol.Spec.InitContainers,
					Containers:                    append(otelCol.Spec.AdditionalContainers, Container(params.Config, params.Log, otelCol, true)),
					Volumes:                       Volumes(params.Config, otelCol),
					DNSPolicy:                     getDNSPolicy(otelCol),
					HostNetwork:                   otelCol.Spec.HostNetwork,
					ShareProcessNamespace:         &otelCol.Spec.ShareProcessNamespace,
					Tolerations:                   otelCol.Spec.Tolerations,
					NodeSelector:                  otelCol.Spec.NodeSelector,
					SecurityContext:               otelCol.Spec.PodSecurityContext,
					PriorityClassName:             otelCol.Spec.PriorityClassName,
					Affinity:                      otelCol.Spec.Affinity,
					TerminationGracePeriodSeconds: otelCol.Spec.TerminationGracePeriodSeconds,
					TopologySpreadConstraints:     otelCol.Spec.TopologySpreadConstraints,
				},
			},
		},
	}, nil
}
