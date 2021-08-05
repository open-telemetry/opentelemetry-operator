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

package reconcilers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/reconcile"
)

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// ConfigMaps reconciles the config map(s) required for the instance in the current context.
func ConfigMaps(ctx context.Context, params reconcile.Params) error {
	desired := []corev1.ConfigMap{
		desiredConfigMap(ctx, params),
	}
	return reconcile.ConfigMaps(ctx, params, desired)
}

func desiredConfigMap(_ context.Context, params reconcile.Params) corev1.ConfigMap {
	name := naming.ConfigMap(params.Instance)
	labels := collector.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = name

	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Data: map[string]string{
			"collector.yaml": params.Instance.Spec.Config,
		},
	}
}
