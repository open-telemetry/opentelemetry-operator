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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;create;list;watch;delete

// Roles reconciles the role(s) required for the instance in the current context.
func Roles(ctx context.Context, params Params) error {
	desired := []rbacv1.Role{}

	if checkEnabled(params) {
		_, err := checkConfig(params)
		if err != nil {
			return fmt.Errorf("failed to parse Prometheus config: %v", err)
		}
		desired = append(desired, desiredRoles(params))
	}

	// first, handle the create/update parts
	if err := expectedRoles(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %v", err)
	}

	// then, delete the extra objects
	if err := deleteRoles(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %v", err)
	}

	return nil
}

func desiredRoles(params Params) rbacv1.Role {
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TARole(params.Instance)

	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TARole(params.Instance),
			Namespace: params.Instance.Namespace,
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		}},
	}
}

func expectedRoles(ctx context.Context, params Params, expected []rbacv1.Role) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &rbacv1.Role{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "role.name", desired.Name, "role.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		params.Log.V(2).Info("applied", "role.name", desired.Name, "role.namespace", desired.Namespace)
	}

	return nil
}

func deleteRoles(ctx context.Context, params Params, expected []rbacv1.Role) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Name, "targetallocator"),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &rbacv1.RoleList{}
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
			params.Log.V(2).Info("deleted", "role.name", existing.Name, "role.namespace", existing.Namespace)
		}
	}

	return nil
}
