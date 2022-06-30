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

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// HorizontalPodAutoscaler reconciles HorizontalPodAutoscalers if autoscale is true and replicas is nil.
func HorizontalPodAutoscalers(ctx context.Context, params Params) error {
	desired := []autoscalingv2beta2.HorizontalPodAutoscaler{}

	// check if autoscale mode is on, e.g MaxReplicas is not nil
	if params.Instance.Spec.AutoScaleSpec.MaxReplicas != nil {
		desired = append(desired, collector.HorizontalPodAutoscaler(params.Config, params.Log, params.Instance))
	}

	// first, handle the create/update parts
	if err := expectedHorizontalPodAutoscalers(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected horizontal pod autoscalers: %w", err)
	}

	// then, delete the extra objects
	if err := deleteHorizontalPodAutoscalers(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the horizontal pod autoscalers: %w", err)
	}

	return nil
}

func expectedHorizontalPodAutoscalers(ctx context.Context, params Params, expected []autoscalingv2beta2.HorizontalPodAutoscaler) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &autoscalingv2beta2.HorizontalPodAutoscaler{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "hpa.name", desired.Name, "hpa.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get %w", err)
		}

		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.OwnerReferences = desired.OwnerReferences
		updated.Spec.MinReplicas = params.Instance.Spec.AutoScaleSpec.MinReplicas
		if params.Instance.Spec.MaxReplicas != nil {
			updated.Spec.MaxReplicas = *params.Instance.Spec.AutoScaleSpec.MaxReplicas
		}

		for k, v := range desired.Annotations {
			updated.Annotations[k] = v
		}
		for k, v := range desired.Labels {
			updated.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "hpa.name", desired.Name, "hpa.namespace", desired.Namespace)
	}

	return nil
}

func deleteHorizontalPodAutoscalers(ctx context.Context, params Params, expected []autoscalingv2beta2.HorizontalPodAutoscaler) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}

	list := &autoscalingv2beta2.HorizontalPodAutoscalerList{}
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
			params.Log.V(2).Info("deleted", "hpa.name", existing.Name, "hpa.namespace", existing.Namespace)
		}
	}

	return nil
}
