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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) appsv1.Deployment {
	labels := Labels(otelcol)
	labels["app.kubernetes.io/name"] = naming.TargetAllocator(otelcol)

	var replicas int32 = 1

	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TargetAllocator(otelcol),
			Namespace: otelcol.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: otelcol.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{Container(cfg, logger, otelcol)},
					Volumes:    Volumes(cfg, otelcol),
				},
			},
		},
	}
}
