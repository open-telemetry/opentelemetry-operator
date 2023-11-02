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

package opampbridge

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
	name := naming.OpAMPBridge(params.OpAMPBridge.Name)
	labels := manifestutils.Labels(params.OpAMPBridge.ObjectMeta, name, params.OpAMPBridge.Spec.Image, ComponentOpAMPBridge, params.Config.LabelsFilter())

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OpAMPBridge.Namespace,
			Labels:      labels,
			Annotations: params.OpAMPBridge.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: params.OpAMPBridge.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.OpAMPBridge.ObjectMeta, ComponentOpAMPBridge),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: params.OpAMPBridge.Spec.PodAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        ServiceAccountName(params.OpAMPBridge),
					Containers:                []corev1.Container{Container(params.Config, params.Log, params.OpAMPBridge)},
					Volumes:                   Volumes(params.Config, params.OpAMPBridge),
					DNSPolicy:                 getDNSPolicy(params.OpAMPBridge),
					HostNetwork:               params.OpAMPBridge.Spec.HostNetwork,
					Tolerations:               params.OpAMPBridge.Spec.Tolerations,
					NodeSelector:              params.OpAMPBridge.Spec.NodeSelector,
					SecurityContext:           params.OpAMPBridge.Spec.PodSecurityContext,
					PriorityClassName:         params.OpAMPBridge.Spec.PriorityClassName,
					Affinity:                  params.OpAMPBridge.Spec.Affinity,
					TopologySpreadConstraints: params.OpAMPBridge.Spec.TopologySpreadConstraints,
				},
			},
		},
	}
}
