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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestDesiredService(t *testing.T) {
	t.Run("should return service with default port", func(t *testing.T) {
		expected := service("test-targetallocator")
		actual := desiredService(params())

		assert.Equal(t, expected, actual)
	})

}

func TestExpectedServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), params(), []corev1.Service{service("targetallocator")})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
}

func TestDeleteServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		deleteService := service("test-delete-targetallocator", 8888)
		createObjectIfNotExists(t, "test-delete-targetallocator", &deleteService)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.True(t, exists)

		err = deleteServices(context.Background(), params(), []corev1.Service{desiredService(params())})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func service(name string, portOpt ...int32) corev1.Service {
	port := int32(443)
	if len(portOpt) > 0 {
		port = portOpt[0]
	}
	params := params()
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TAService(params.Instance)

	selector := targetallocator.Labels(params.Instance)
	selector["app.kubernetes.io/name"] = naming.TargetAllocator(params.Instance)

	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.Instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:       "targetallocation",
				Port:       port,
				TargetPort: intstr.FromInt(443),
			}},
		},
	}
}
