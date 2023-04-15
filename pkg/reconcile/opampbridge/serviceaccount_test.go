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

package opampbridge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/opampbridge"
)

func TestExpectedServiceAccounts(t *testing.T) {
	t.Run("should create service account", func(t *testing.T) {
		opampBridgeDesired := opampbridge.ServiceAccount(params().Instance)
		err := expectedServiceAccounts(context.Background(), params(), []v1.ServiceAccount{opampBridgeDesired})
		assert.NoError(t, err)

		opampBridgeExists, err := populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, opampBridgeExists)

	})

	t.Run("should update existing service account", func(t *testing.T) {
		existing := v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-opamp-bridge",
				Namespace: "default",
			},
		}
		createObjectIfNotExists(t, "test-opamp-bridge", &existing)
		exists, err := populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = expectedServiceAccounts(context.Background(), params(), []v1.ServiceAccount{opampbridge.ServiceAccount(params().Instance)})
		assert.NoError(t, err)

		actual := v1.ServiceAccount{}
		_, err = populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})
		assert.NoError(t, err)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
	})
}

func TestDeleteServiceAccounts(t *testing.T) {
	t.Run("should delete the managed service account", func(t *testing.T) {
		existing := v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-opamp-bridge",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-opamp-bridge", &existing)
		exists, err := populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = deleteServiceAccounts(context.Background(), params(), []v1.ServiceAccount{opampbridge.ServiceAccount(params().Instance)})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.NoError(t, err)
		assert.False(t, exists)
	})
	t.Run("should not delete unrelated service account", func(t *testing.T) {
		existing := v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-opamp-bridge",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.testing",
					"app.kubernetes.io/managed-by": "helm-opentelemetry",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-opamp-bridge", &existing)
		exists, err := populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = deleteServiceAccounts(context.Background(), params(), []v1.ServiceAccount{opampbridge.ServiceAccount(params().Instance)})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, types.NamespacedName{Namespace: "default", Name: "test-delete-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, exists)

	})

}

func TestDesiredServiceAccounts(t *testing.T) {
	t.Run("should not create any service account", func(t *testing.T) {
		params := params()
		params.Instance.Spec.ServiceAccount = "existing-opamp-bridge-sa"
		desired := desiredServiceAccounts(params)
		assert.Len(t, desired, 0)
	})
	t.Run("should create opamp-bridge service account", func(t *testing.T) {
		params := params()
		desired := desiredServiceAccounts(params)
		assert.Len(t, desired, 1)
		assert.Equal(t, opampbridge.ServiceAccount(params.Instance), desired[0])
	})
}
