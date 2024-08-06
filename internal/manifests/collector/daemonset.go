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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// DaemonSet builds the deployment for the given instance.
func DaemonSet(params manifests.Params) (*appsv1.DaemonSet, error) {
	name := naming.Collector(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	podAnnotations, err := manifestutils.PodAnnotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	imagePullSecrets := params.OtelCol.Spec.ImagePullSecrets
	if params.OtelCol.Spec.ImagePullSecrets == nil {
		imagePullSecrets = []v1.LocalObjectReference{}
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Collector(params.OtelCol.Name),
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:    ServiceAccountName(params.OtelCol),
					ImagePullSecrets:      imagePullSecrets,
					InitContainers:        params.OtelCol.Spec.InitContainers,
					Containers:            append(params.OtelCol.Spec.AdditionalContainers, Container(params.Config, params.Log, params.OtelCol, true)),
					Volumes:               Volumes(params.Config, params.OtelCol),
					Tolerations:           params.OtelCol.Spec.Tolerations,
					NodeSelector:          params.OtelCol.Spec.NodeSelector,
					HostNetwork:           params.OtelCol.Spec.HostNetwork,
					ShareProcessNamespace: &params.OtelCol.Spec.ShareProcessNamespace,
					DNSPolicy:             manifestutils.GetDNSPolicy(params.OtelCol.Spec.HostNetwork, params.OtelCol.Spec.PodDNSConfig),
					DNSConfig:             &params.OtelCol.Spec.PodDNSConfig,
					SecurityContext:       params.OtelCol.Spec.PodSecurityContext,
					PriorityClassName:     params.OtelCol.Spec.PriorityClassName,
					Affinity:              params.OtelCol.Spec.Affinity,
				},
			},
			UpdateStrategy: params.OtelCol.Spec.DaemonSetUpdateStrategy,
		},
	}, nil
}
