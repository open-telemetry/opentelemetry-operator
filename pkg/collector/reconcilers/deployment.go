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

	appsv1 "k8s.io/api/apps/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/reconcile"
)

// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Deployments reconciles the deployment(s) required for the instance in the current context.
func Deployments(ctx context.Context, params reconcile.Params) error {
	desired := []appsv1.Deployment{}
	if params.Instance.Spec.Mode == "deployment" {
		desired = append(desired, collector.Deployment(params.Config, params.Log, params.Instance))
	}

	return reconcile.Deployments(ctx, params, desired)
}
