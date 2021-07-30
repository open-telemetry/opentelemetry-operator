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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestDesiredRole(t *testing.T) {
	t.Run("should return role", func(t *testing.T) {
		expected := role("test-view")
		actual := desiredRoles(params())

		assert.Equal(t, expected, actual)
	})

}

func TestExpectedRoles(t *testing.T) {
	t.Run("should create the role", func(t *testing.T) {
		err := expectedRoles(context.Background(), params(), []rbacv1.Role{role("view")})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &rbacv1.Role{}, types.NamespacedName{Namespace: "default", Name: "view"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
}

func TestDeleteRoles(t *testing.T) {
	t.Run("should delete excess roles", func(t *testing.T) {
		deleteRole := role("test-delete-view")
		createObjectIfNotExists(t, "test-delete-view", &deleteRole)

		exists, err := populateObjectIfExists(t, &rbacv1.Role{}, types.NamespacedName{Namespace: "default", Name: "test-delete-view"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = deleteRoles(context.Background(), params(), []rbacv1.Role{desiredRoles(params())})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &rbacv1.Role{}, types.NamespacedName{Namespace: "default", Name: "test-delete-view"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func role(name string) rbacv1.Role {
	params := params()
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TARole(params.Instance)

	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
