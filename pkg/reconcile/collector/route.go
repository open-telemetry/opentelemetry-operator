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
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"

	routev1 "github.com/openshift/api/route/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// Routes reconciles the route(s) required for the instance in the current context.
// TODO: This functionality should be put with the rest of reconciliation logic in the mutate.go
// https://github.com/open-telemetry/opentelemetry-operator/issues/2108
func Routes(ctx context.Context, params manifests.Params) error {
	if params.Instance.Spec.Ingress.Type != v1alpha1.IngressTypeRoute {
		return nil
	}

	isSupportedMode := true
	if params.Instance.Spec.Mode == v1alpha1.ModeSidecar {
		params.Log.V(3).Info("ingress settings are not supported in sidecar mode")
		isSupportedMode = false
	}

	var desired []*routev1.Route
	if isSupportedMode {
		if r := collector.Routes(params.Config, params.Log, params.Instance); r != nil {
			desired = append(desired, r...)
		}
	}

	// first, handle the create/update parts
	if err := expectedRoutes(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected routes: %w", err)
	}

	// then, delete the extra objects
	if err := deleteRoutes(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the routes to be deleted: %w", err)
	}

	return nil
}

func expectedRoutes(ctx context.Context, params manifests.Params, expected []*routev1.Route) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &routev1.Route{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err = params.Client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "route.name", desired.Name, "route.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
		updated.Spec.To = desired.Spec.To
		updated.Spec.TLS = desired.Spec.TLS
		updated.Spec.Port = desired.Spec.Port
		updated.Spec.WildcardPolicy = desired.Spec.WildcardPolicy

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "route.name", desired.Name, "route.namespace", desired.Namespace)
	}
	return nil
}

func deleteRoutes(ctx context.Context, params manifests.Params, expected []*routev1.Route) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &routev1.RouteList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
				break
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "route.name", existing.Name, "route.namespace", existing.Namespace)
		}
	}

	return nil
}
