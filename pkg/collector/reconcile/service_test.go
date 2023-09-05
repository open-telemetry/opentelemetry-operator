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

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestExpectedServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), params(), []*v1.Service{service("test-collector", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update service", func(t *testing.T) {
		serviceInstance := service("test-collector", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-collector", serviceInstance)

		extraPorts := v1.ServicePort{
			Name:       "port-web",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		}

		ports := append(params().Instance.Spec.Ports, extraPorts)
		err := expectedServices(context.Background(), params(), []*v1.Service{service("test-collector", ports)})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Contains(t, actual.Spec.Ports, extraPorts)
	})
	t.Run("should update service on version change", func(t *testing.T) {
		serviceInstance := service("test-collector", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-collector", serviceInstance)

		newService := service("test-collector", params().Instance.Spec.Ports)
		newService.Spec.Selector["app.kubernetes.io/version"] = "Newest"
		err := expectedServices(context.Background(), params(), []*v1.Service{newService})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, "Newest", actual.Spec.Selector["app.kubernetes.io/version"])
	})
	t.Run("should update service on internal traffic policy change", func(t *testing.T) {
		serviceInstance := service("test-collector", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-collector", serviceInstance)

		newService := serviceWithInternalTrafficPolicy("test-collector", params().Instance.Spec.Ports, v1.ServiceInternalTrafficPolicyLocal)
		err := expectedServices(context.Background(), params(), []*v1.Service{newService})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, v1.ServiceInternalTrafficPolicyLocal, *actual.Spec.InternalTrafficPolicy)
	})
}

func TestDeleteServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		ports := []v1.ServicePort{{
			Port: 80,
			Name: "web",
		}}
		deleteService := service("delete-service-collector", ports)
		createObjectIfNotExists(t, "delete-service-collector", deleteService)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.True(t, exists)

		param := params()
		desired := collector.Service(param.Config, param.Log, param.Instance)
		err = deleteServices(context.Background(), params(), []*v1.Service{desired})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func service(name string, ports []v1.ServicePort) *v1.Service {
	return serviceWithInternalTrafficPolicy(name, ports, v1.ServiceInternalTrafficPolicyCluster)
}

func serviceWithInternalTrafficPolicy(name string, ports []v1.ServicePort, internalTrafficPolicy v1.ServiceInternalTrafficPolicyType) *v1.Service {
	labels := collector.Labels(params().Instance, name, []string{})

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: params().Instance.Annotations,
		},
		Spec: v1.ServiceSpec{
			InternalTrafficPolicy: &internalTrafficPolicy,
			Selector:              collector.SelectorLabels(params().Instance),
			ClusterIP:             "",
			Ports:                 ports,
		},
	}
}
