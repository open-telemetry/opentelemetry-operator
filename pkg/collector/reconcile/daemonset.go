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

package reconcile

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// +kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

// DaemonSets reconciles the daemon set(s) required for the instance in the current context.
func DaemonSets(ctx context.Context, params Params) error {
	desired := []appsv1.DaemonSet{}
	if params.Instance.Spec.Mode == "daemonset" {
		desired = append(desired, collector.DaemonSet(params.Config, params.Log, params.Instance))
	}

	// first, handle the create/update parts
	if err := expectedDaemonSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected daemon sets: %w", err)
	}

	// then, delete the extra objects
	if err := deleteDaemonSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the daemon sets to be deleted: %w", err)
	}

	return nil
}

func expectedDaemonSets(ctx context.Context, params Params, expected []appsv1.DaemonSet) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &appsv1.DaemonSet{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "daemonset.name", desired.Name, "daemonset.namespace", desired.Namespace)
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

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		// Selector is an immutable field, if set, we cannot modify it otherwise we will face reconciliation error.
		updated.Spec.Selector = existing.Spec.Selector.DeepCopy()

		patch := client.MergeFrom(existing)
		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "daemonset.name", desired.Name, "daemonset.namespace", desired.Namespace)
	}

	return nil
}

func deleteDaemonSets(ctx context.Context, params Params, expected []appsv1.DaemonSet) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.DaemonSetList{}
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
			params.Log.V(2).Info("deleted", "daemonset.name", existing.Name, "daemonset.namespace", existing.Namespace)
		}
	}

	return nil
}
