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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
)

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Services reconciles the service(s) required for the instance in the current context.
func Services(ctx context.Context, params manifests.Params) error {
	var desired []*corev1.Service
	if params.Instance.Spec.Mode != v1alpha1.ModeSidecar {
		type builder func(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.Service
		for _, builder := range []builder{collector.Service, collector.HeadlessService, collector.MonitoringService} {
			svc := builder(params.Config, params.Log, params.Instance)
			// add only the non-nil to the list
			if svc != nil {
				desired = append(desired, svc)
			}
		}
	}

	if params.Instance.Spec.TargetAllocator.Enabled {
		desired = append(desired, targetallocator.Service(params.Config, params.Log, params.Instance))
	}

	// first, handle the create/update parts
	if err := expectedServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %w", err)
	}

	// then, delete the extra objects
	if err := deleteServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %w", err)
	}

	return nil
}

func expectedServices(ctx context.Context, params manifests.Params, expected []*corev1.Service) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &corev1.Service{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if clientErr := params.Client.Create(ctx, desired); clientErr != nil {
				return fmt.Errorf("failed to create: %w", clientErr)
			}
			params.Log.V(2).Info("created", "service.name", desired.Name, "service.namespace", desired.Namespace)
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

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}
		updated.Spec.Ports = desired.Spec.Ports
		updated.Spec.Selector = desired.Spec.Selector

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "service.name", desired.Name, "service.namespace", desired.Namespace)
	}

	return nil
}

func deleteServices(ctx context.Context, params manifests.Params, expected []*corev1.Service) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &corev1.ServiceList{}
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
			params.Log.V(2).Info("deleted", "service.name", existing.Name, "service.namespace", existing.Namespace)
		}
	}

	return nil
}
