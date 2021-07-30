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

	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;create;list;watch;delete

// RoleBindings reconciles the role binding(s) required for the instance in the current context.
func RoleBindings(ctx context.Context, params Params) error {
	desired := []rbacv1.RoleBinding{}

	if checkEnabled(params) {
		_, err := checkConfig(params)
		if err != nil {
			return fmt.Errorf("failed to parse Prometheus config: %v", err)
		}
		desired = append(desired, desiredRoleBinding(params))
	}

	// first, handle the create/update parts
	if err := expectedRoleBindings(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %v", err)
	}

	// then, delete the extra objects
	if err := deleteRoleBindings(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %v", err)
	}

	return nil
}

func desiredRoleBinding(params Params) rbacv1.RoleBinding {
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TARoleBinding(params.Instance)

	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TARoleBinding(params.Instance),
			Namespace: params.Instance.Namespace,
			Labels:    labels,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "default",
			Namespace: params.Instance.Namespace,
		}},
		RoleRef: v1.RoleRef{
			Kind: "Role",
			Name: naming.TARole(params.Instance),
		},
	}
}

func expectedRoleBindings(ctx context.Context, params Params, expected []rbacv1.RoleBinding) error {
	for _, obj := range expected {
		desired := obj

		existing := &rbacv1.RoleBinding{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "rolebinding.name", desired.Name, "rolebinding.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		params.Log.V(2).Info("applied", "rolebinding.name", desired.Name, "rolebinding.namespace", desired.Namespace)
	}

	return nil
}

func deleteRoleBindings(ctx context.Context, params Params, expected []rbacv1.RoleBinding) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Name, "targetallocator"),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &rbacv1.RoleBindingList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
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
			params.Log.V(2).Info("deleted", "rolebinding.name", existing.Name, "rolebinding.namespace", existing.Namespace)
		}
	}

	return nil
}
