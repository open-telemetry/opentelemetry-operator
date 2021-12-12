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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// ServiceAccounts reconciles the service account(s) required for the instance in the current context
func ServiceAccounts(ctx context.Context, params Params) error {
	desired := []corev1.ServiceAccount{
		desiredServiceAccount(ctx, params),
	}

	// first, handle the create/update parts
	if err := expectedServiceAccounts(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected service accounts: %v", err)
	}

	// then, delete the extra objects
	if err := deleteServiceAccounts(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the service accounts to be deleted: %v", err)
	}

	return nil
}

func desiredServiceAccount(ctx context.Context, params Params) corev1.ServiceAccount {
	name := fmt.Sprintf("%s-collector", params.Instance.Name)

	labels := collector.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = name

	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
	}
}

func expectedServiceAccounts(ctx context.Context, params Params, expected []corev1.ServiceAccount) error {
	for _, obj := range expected {
		desired := obj

		controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme)

		existing := &corev1.ServiceAccount{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "serviceaccount.name", desired.Name, "serviceaccount.namespace", desired.Namespace)
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

		if err := params.Client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "serviceaccount.name", desired.Name, "serviceaccount.namespace", desired.Namespace)
	}

	return nil
}

func deleteServiceAccounts(ctx context.Context, params Params, expected []corev1.ServiceAccount) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &corev1.ServiceAccountList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for _, existing := range list.Items {
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "serviceaccount.name", existing.Name, "serviceaccount.namespace", existing.Namespace)
		}
	}

	return nil
}

// ServiceAccountNameFor returns the name of the service account for the given context
func ServiceAccountNameFor(instance v1alpha1.OpenTelemetryCollector) string {
	if len(instance.Spec.ServiceAccount) == 0 {
		return fmt.Sprintf("%s-collector", instance.Name)
	}

	return instance.Spec.ServiceAccount
}
