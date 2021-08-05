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

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/reconcile"
)

// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// ServiceAccounts reconciles the service account(s) required for the instance in the current context.
func ServiceAccounts(ctx context.Context, params reconcile.Params) error {
	desired := []corev1.ServiceAccount{}
	if params.Instance.Spec.Mode != v1alpha1.ModeSidecar {
		desired = append(desired, collector.ServiceAccount(params.Instance))
	}

	return reconcile.ServiceAccounts(ctx, params, desired)
}
