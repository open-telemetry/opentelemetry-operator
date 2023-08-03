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

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// ServiceMonitors reconciles the service monitor(s) required for the instance in the current context.
func ServiceMonitors(ctx context.Context, params manifests.Params) error {
	if !featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		return nil
	}

	var desired []*monitoringv1.ServiceMonitor

	if params.Instance.Spec.Observability.Metrics.EnableMetrics {
		if sm, err := collector.ServiceMonitor(params.Config, params.Log, params.Instance); err != nil {
			return err
		} else {
			desired = append(desired, sm)
		}
	}

	desired = append(desired, collector.ServiceMonitorFromConfig(params.Config, params.Log, params.Instance)...)

	// first, handle the create/update parts
	if err := expectedServiceMonitors(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected service monitors: %w", err)
	}

	// then, delete the extra objects
	if err := deleteServiceMonitors(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the service monitors to be deleted: %w", err)
	}

	return nil
}

func expectedServiceMonitors(ctx context.Context, params manifests.Params, expected []*monitoringv1.ServiceMonitor) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &monitoringv1.ServiceMonitor{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err = params.Client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "servicemonitor.name", desired.Name, "servicemonitor.namespace", desired.Namespace)
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
		updated.Spec.Endpoints = desired.Spec.Endpoints
		updated.Spec.NamespaceSelector = desired.Spec.NamespaceSelector
		updated.Spec.Selector = desired.Spec.Selector

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

		params.Log.V(2).Info("applied", "servicemonitor.name", desired.Name, "servicemonitor.namespace", desired.Namespace)
	}
	return nil
}

func deleteServiceMonitors(ctx context.Context, params manifests.Params, expected []*monitoringv1.ServiceMonitor) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}

	list := &monitoringv1.ServiceMonitorList{}
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
			if err := params.Client.Delete(ctx, existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "servicemonitor.name", existing.Name, "servicemonitor.namespace", existing.Namespace)
		}
	}

	return nil
}
