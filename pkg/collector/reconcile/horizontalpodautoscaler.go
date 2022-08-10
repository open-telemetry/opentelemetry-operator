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

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// HorizontalPodAutoscaler reconciles HorizontalPodAutoscalers if autoscale is true and replicas is nil.
func HorizontalPodAutoscalers(ctx context.Context, params Params) error {
	desired := []runtime.Object{}

	// check if autoscale mode is on, e.g MaxReplicas is not nil
	if params.Instance.Spec.MaxReplicas != nil {
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

func expectedHorizontalPodAutoscalers(ctx context.Context, params Params, expected []runtime.Object) error {
	autoscalingVersion := params.Config.AutoscalingVersion()
	var existing client.Object
	if autoscalingVersion == config.AutoscalingVersionV2Beta2 {
		existing = &autoscalingv2beta2.HorizontalPodAutoscaler{}
	} else {
		existing = &autoscalingv2.HorizontalPodAutoscaler{}
	}

	for _, obj := range expected {
		desired, _ := meta.Accessor(obj)

		if err := controllerutil.SetControllerReference(&params.Instance, desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		nns := types.NamespacedName{Namespace: desired.GetNamespace(), Name: desired.GetName()}
		err := params.Client.Get(ctx, nns, existing)
		if k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, obj.(client.Object)); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "hpa.name", desired.GetName(), "hpa.namespace", desired.GetNamespace())
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get %w", err)
		}

		existing.SetOwnerReferences(desired.GetOwnerReferences())
		setAutoscalerSpec(autoscalingVersion, params.Instance, existing)

		annotations := existing.GetAnnotations()
		for k, v := range desired.GetAnnotations() {
			annotations[k] = v
		}
		existing.SetAnnotations(annotations)
		labels := existing.GetLabels()
		for k, v := range desired.GetLabels() {
			labels[k] = v
		}
		existing.SetLabels(labels)

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, existing, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "hpa.name", desired.GetName(), "hpa.namespace", desired.GetNamespace())
	}

	return nil
}

func setAutoscalerSpec(autoscalingVersion string, instance v1alpha1.OpenTelemetryCollector, autoscaler runtime.Object) {
	one := int32(1)
	if autoscalingVersion == config.AutoscalingVersionV2Beta2 {
		existing := autoscaler.(*autoscalingv2beta2.HorizontalPodAutoscaler)
		if instance.Spec.MaxReplicas != nil {
			existing.Spec.MaxReplicas = *instance.Spec.MaxReplicas
			if instance.Spec.MinReplicas != nil {
				existing.Spec.MinReplicas = instance.Spec.MinReplicas
			} else {
				existing.Spec.MinReplicas = &one
			}
		}
	} else {
		existing := autoscaler.(*autoscalingv2.HorizontalPodAutoscaler)
		if instance.Spec.MaxReplicas != nil {
			existing.Spec.MaxReplicas = *instance.Spec.MaxReplicas
			if instance.Spec.MinReplicas != nil {
				existing.Spec.MinReplicas = instance.Spec.MinReplicas
			} else {
				existing.Spec.MinReplicas = &one
			}
		}
	}
}

func deleteHorizontalPodAutoscalers(ctx context.Context, params Params, expected []runtime.Object) error {
	autoscalingVersion := params.Config.AutoscalingVersion()

	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}

	if autoscalingVersion == config.AutoscalingVersionV2Beta2 {
		list := &autoscalingv2beta2.HorizontalPodAutoscalerList{}
		if err := params.Client.List(ctx, list, opts...); err != nil {
			return fmt.Errorf("failed to list: %w", err)
		}

		for i := range list.Items {
			existing := list.Items[i]
			del := true
			for _, k := range expected {
				keep := k.(*autoscalingv2beta2.HorizontalPodAutoscaler)
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
	} else {
		list := &autoscalingv2.HorizontalPodAutoscalerList{}
		if err := params.Client.List(ctx, list, opts...); err != nil {
			return fmt.Errorf("failed to list: %w", err)
		}

		for i := range list.Items {
			existing := list.Items[i]
			del := true
			for _, k := range expected {
				keep := k.(*autoscalingv2.HorizontalPodAutoscaler)
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
	}

	return nil
}
