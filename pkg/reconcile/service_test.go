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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestExpectedServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), params(), []v1.Service{service("test-collector", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update service", func(t *testing.T) {
		serviceInstance := service("test-collector", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-collector", &serviceInstance)

		extraPorts := v1.ServicePort{
			Name:       "port-web",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		}

		ports := append(params().Instance.Spec.Ports, extraPorts)
		err := expectedServices(context.Background(), params(), []v1.Service{service("test-collector", ports)})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Contains(t, actual.Spec.Ports, extraPorts)

	})
}

func TestDeleteServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		ports := []v1.ServicePort{{
			Port: 80,
			Name: "web",
		}}
		deleteService := service("delete-service-collector", ports)
		createObjectIfNotExists(t, "delete-service-collector", &deleteService)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.True(t, exists)

		desired := service("desired-service-collector", ports)
		err = deleteServices(context.Background(), params(), []v1.Service{desired})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func service(name string, ports []v1.ServicePort) v1.Service {
	labels := collector.Labels(params().Instance)
	labels["app.kubernetes.io/name"] = name

	selector := labels
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: params().Instance.Annotations,
		},
		Spec: v1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports:     ports,
		},
	}
}
