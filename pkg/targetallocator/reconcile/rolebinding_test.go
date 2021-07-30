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
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestDesiredRoleBinding(t *testing.T) {
	t.Run("should return role binding", func(t *testing.T) {
		expected := roleBinding("test-targetallocator")
		actual := desiredRoleBinding(params())

		assert.Equal(t, expected, actual)
	})

}

func TestExpectedRoleBindings(t *testing.T) {
	t.Run("should create the role binding", func(t *testing.T) {
		err := expectedRoleBindings(context.Background(), params(), []rbacv1.RoleBinding{roleBinding("targetallocator")})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &rbacv1.RoleBinding{}, types.NamespacedName{Namespace: "default", Name: "targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
}

func TestDeleteRoleBindings(t *testing.T) {
	t.Run("should delete excess role bindings", func(t *testing.T) {
		deleteRB := roleBinding("test-delete-targetallocator")
		createObjectIfNotExists(t, "test-delete-targetallocator", &deleteRB)

		exists, err := populateObjectIfExists(t, &rbacv1.RoleBinding{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = deleteRoleBindings(context.Background(), params(), []rbacv1.RoleBinding{desiredRoleBinding(params())})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &rbacv1.RoleBinding{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func roleBinding(name string) rbacv1.RoleBinding {
	params := params()
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TARoleBinding(params.Instance)

	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
