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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create config map", func(t *testing.T) {
		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("desired-configmap")}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "desired-configmap"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update config map", func(t *testing.T) {
		cm := configMap("desired-configmap")
		createObjectIfNotExists(t, "desired-configmap", &cm)

		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("desired-configmap")}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "desired-configmap"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
	})

	t.Run("should delete config map", func(t *testing.T) {
		deletecm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delete-collector",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		}
		createObjectIfNotExists(t, "test-delete-collector", &deletecm)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-collector"})
		assert.True(t, exists)

		err := deleteConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("desired-configmap")})
		assert.NoError(t, err)

		exists, _ = populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-delete-collector"})
		assert.False(t, exists)
	})
}

func configMap(name string) v1.ConfigMap {
	labels := map[string]string{}
	labels["app.kubernetes.io/name"] = name

	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
	}
}
