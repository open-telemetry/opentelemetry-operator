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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/opampbridge"
)

func TestDesiredService(t *testing.T) {
	t.Run("should return service with port mentioned in Instance.Spec.Ports", func(t *testing.T) {

		opampBridgePort := v1.ServicePort{
			Name:       "opamp-bridge",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
		}

		ports := append(params().Instance.Spec.Ports, opampBridgePort)
		expected := service("test-opamp-bridge", ports)
		actual := desiredService(context.Background(), params())

		assert.Equal(t, expected, *actual)
	})
}

func TestExpectedServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), params(), []v1.Service{service("test-opamp-bridge", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update service", func(t *testing.T) {
		serviceInstance := service("test-opamp-bridge", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-opamp-bridge", &serviceInstance)

		extraPort := v1.ServicePort{
			Name:       "port-web",
			Protocol:   "TCP",
			Port:       3030,
			TargetPort: intstr.FromInt(3030),
		}

		ports := append(params().Instance.Spec.Ports, extraPort)
		err := expectedServices(context.Background(), params(), []v1.Service{service("test-opamp-bridge", ports)})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Contains(t, actual.Spec.Ports, extraPort)
	})
	t.Run("should update service on version change", func(t *testing.T) {
		serviceInstance := service("test-opamp-bridge", params().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-opamp-bridge", &serviceInstance)

		newService := service("test-opamp-bridge", params().Instance.Spec.Ports)
		newService.Spec.Selector["app.kubernetes.io/version"] = "Newest"
		err := expectedServices(context.Background(), params(), []v1.Service{newService})
		assert.NoError(t, err)

		actual := v1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-opamp-bridge"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, "Newest", actual.Spec.Selector["app.kubernetes.io/version"])
	})
}

func TestDeleteServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		ports := []v1.ServicePort{{
			Port: 8081,
			Name: "web",
		}}
		deleteService := service("delete-service-opamp-bridge", ports)
		createObjectIfNotExists(t, "delete-service-opamp-bridge", &deleteService)

		exists, err := populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-opamp-bridge"})
		assert.NoError(t, err)
		assert.True(t, exists)

		desired := desiredService(context.Background(), params())
		err = deleteServices(context.Background(), params(), []v1.Service{*desired})
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-opamp-bridge"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func service(name string, ports []v1.ServicePort) v1.Service {
	labels := opampbridge.Labels(params().Instance, []string{})
	labels["app.kubernetes.io/name"] = name

	selectorLabels := opampbridge.SelectorLabels(params().Instance)
	selectorLabels["app.kubernetes.io/name"] = naming.OpAMPBridge(params().Instance)

	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: params().Instance.Annotations,
		},
		Spec: v1.ServiceSpec{
			Selector:  selectorLabels,
			ClusterIP: "",
			Ports:     ports,
		},
	}
}
