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
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expected := configMap("test-collector")
		actual := desiredConfigMap(context.Background(), params())
		assert.Equal(t, expected, actual)
	})

}

func TestExpectedConfigMap(t *testing.T) {

	cm := configMap("test-collector")
	deletecm := configMap("test")

	t.Run("should create config map", func(t *testing.T) {
		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{cm}, true)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update config map", func(t *testing.T) {

		createObjectIfNotExists(t, "test-collector", &cm)

		//err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{desiredConfigMap(context.Background(), params())}, true)
		//assert.NoError(t, err)
		//
		//actual := v1.ConfigMap{}
		//exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		//
		//assert.NoError(t, err)
		//assert.True(t, exists)
		//assert.Equal(t, actual.Data, cm.Data)
	})

	t.Run("should delete config map", func(t *testing.T) {
		createObjectIfNotExists(t, "test", &deletecm)
		err := deleteConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("test-collector")})
		assert.NoError(t, err)

		exists, _ := populateObjectIfExists(t, &v1.ConfigMap{}, types.NamespacedName{Namespace: "default", Name: "test"})

		assert.False(t, exists)
	})
}

func configMap(name string) v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params().Instance.Namespace, params().Instance.Name),
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-collector",
				"app.kubernetes.io/name":       name,
			},
		},
		Data: map[string]string{
			"collector.yaml": params().Instance.Spec.Config,
		},
	}
}
