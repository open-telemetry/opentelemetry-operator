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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// +kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// StatefulSets reconciles the stateful set(s) required for the instance in the current context.
func StatefulSets(ctx context.Context, params Params) error {

	desired := []appsv1.StatefulSet{}
	if params.Instance.Spec.Mode == "statefulset" {
		desired = append(desired, collector.StatefulSet(params.Config, params.Log, params.Instance))
	}

	// first, handle the create/update parts
	if err := expectedStatefulSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected stateful sets: %w", err)
	}

	// then, delete the extra objects
	if err := deleteStatefulSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the stateful sets to be deleted: %w", err)
	}

	return nil
}

func expectedStatefulSets(ctx context.Context, params Params, expected []appsv1.StatefulSet) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &appsv1.StatefulSet{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if clientErr := params.Client.Create(ctx, &desired); clientErr != nil {
				return fmt.Errorf("failed to create: %w", clientErr)
			}
			params.Log.V(2).Info("created", "statefulset.name", desired.Name, "statefulset.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// Check for immutable fields. If set, we cannot modify the stateful set, otherwise we will face reconciliation error.
		if needsDeletion, fieldName := hasImmutableFieldChange(&desired, existing); needsDeletion {
			params.Log.V(2).Info("Immutable field change detected, trying to delete, the new collector statefulset will be created in the next reconcile cycle",
				"field", fieldName, "statefulset.name", existing.Name, "statefulset.namespace", existing.Namespace)

			if err := params.Client.Delete(ctx, existing); err != nil {
				return fmt.Errorf("failed to delete statefulset: %w", err)
			}
			continue
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

		patch := client.MergeFrom(existing)
		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "statefulset.name", desired.Name, "statefulset.namespace", desired.Namespace)
	}

	return nil
}

func deleteStatefulSets(ctx context.Context, params Params, expected []appsv1.StatefulSet) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.StatefulSetList{}
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
			params.Log.V(2).Info("deleted", "statefulset.name", existing.Name, "statefulset.namespace", existing.Namespace)
		}
	}

	return nil
}

func hasImmutableFieldChange(desired, existing *appsv1.StatefulSet) (bool, string) {
	if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
		return true, "Spec.Selector"
	}

	if hasVolumeClaimsTemplatesChanged(desired, existing) {
		return true, "Spec.VolumeClaimTemplates"
	}

	return false, ""
}

// hasVolumeClaimsTemplatesChanged if volume claims template change has been detected.
// We need to do this manually due to some fields being automatically filled by the API server
// and these needs to be excluded from the comparison to prevent false positives.
func hasVolumeClaimsTemplatesChanged(desired, existing *appsv1.StatefulSet) bool {
	if len(desired.Spec.VolumeClaimTemplates) != len(existing.Spec.VolumeClaimTemplates) {
		return true
	}

	for i := range desired.Spec.VolumeClaimTemplates {
		// VolumeMode is automatically set by the API server, so if it is not set in the CR, assume it's the same as the existing one.
		if desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode == nil || *desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode == "" {
			desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode = existing.Spec.VolumeClaimTemplates[i].Spec.VolumeMode
		}

		if desired.Spec.VolumeClaimTemplates[i].Name != existing.Spec.VolumeClaimTemplates[i].Name {
			return true
		}
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Annotations, existing.Spec.VolumeClaimTemplates[i].Annotations) {
			return true
		}
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Spec, existing.Spec.VolumeClaimTemplates[i].Spec) {
			return true
		}
	}

	return false
}
