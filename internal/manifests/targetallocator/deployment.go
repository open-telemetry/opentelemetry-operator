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

package targetallocator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(params manifests.Params) (*appsv1.Deployment, error) {
	name := naming.TargetAllocator(params.OtelCol.Name)
	labels := Labels(params.OtelCol, name)

	configMap, err := ConfigMap(params)
	if err != nil {
		params.Log.Info("failed to construct target allocator config map for annotations")
		configMap = nil
	}
	annotations := Annotations(params.OtelCol, configMap)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.OtelCol.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: params.OtelCol.Spec.TargetAllocator.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        ServiceAccountName(params.OtelCol),
					Containers:                []corev1.Container{Container(params.Config, params.Log, params.OtelCol)},
					Volumes:                   Volumes(params.Config, params.OtelCol),
					NodeSelector:              params.OtelCol.Spec.TargetAllocator.NodeSelector,
					Tolerations:               params.OtelCol.Spec.TargetAllocator.Tolerations,
					TopologySpreadConstraints: params.OtelCol.Spec.TargetAllocator.TopologySpreadConstraints,
					Affinity:                  params.OtelCol.Spec.TargetAllocator.Affinity,
					SecurityContext:           params.OtelCol.Spec.TargetAllocator.SecurityContext,
				},
			},
		},
	}, nil
}
