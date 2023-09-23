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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(cfg config.Config, logger logr.Logger, opampBridge v1alpha1.OpAMPBridge) appsv1.Deployment {
	name := naming.OpAMPBridge(opampBridge.Name)
	labels := Labels(opampBridge, name, cfg.LabelsFilter())

	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opampBridge.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: opampBridge.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: SelectorLabels(opampBridge),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: opampBridge.Spec.PodAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        ServiceAccountName(opampBridge),
					Containers:                []corev1.Container{Container(cfg, logger, opampBridge)},
					Volumes:                   Volumes(cfg, opampBridge),
					DNSPolicy:                 getDNSPolicy(opampBridge),
					HostNetwork:               opampBridge.Spec.HostNetwork,
					Tolerations:               opampBridge.Spec.Tolerations,
					NodeSelector:              opampBridge.Spec.NodeSelector,
					SecurityContext:           opampBridge.Spec.PodSecurityContext,
					PriorityClassName:         opampBridge.Spec.PriorityClassName,
					Affinity:                  opampBridge.Spec.Affinity,
					TopologySpreadConstraints: opampBridge.Spec.TopologySpreadConstraints,
				},
			},
		},
	}
}
