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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestExtractPortNumbersAndNames(t *testing.T) {
	t.Run("should return extracted port names and numbers", func(t *testing.T) {
		ports := []corev1.ServicePort{{Name: "web", Port: 8080}, {Name: "tcp", Port: 9200}}
		expectedPortNames := map[string]bool{"web": true, "tcp": true}
		expectedPortNumbers := map[int32]bool{8080: true, 9200: true}

		actualPortNumbers, actualPortNames := extractPortNumbersAndNames(ports)
		assert.Equal(t, expectedPortNames, actualPortNames)
		assert.Equal(t, expectedPortNumbers, actualPortNumbers)

	})
}

func TestFilterPort(t *testing.T) {

	tests := []struct {
		name        string
		candidate   corev1.ServicePort
		portNumbers map[int32]bool
		portNames   map[string]bool
		expected    corev1.ServicePort
	}{
		{
			name:        "should filter out duplicate port",
			candidate:   corev1.ServicePort{Name: "web", Port: 8080},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"test": true, "metrics": true},
		},

		{
			name:        "should not filter unique port",
			candidate:   corev1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"test": true, "metrics": true},
			expected:    corev1.ServicePort{Name: "web", Port: 8090},
		},

		{
			name:        "should change the duplicate portName",
			candidate:   corev1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"web": true, "metrics": true},
			expected:    corev1.ServicePort{Name: "port-8090", Port: 8090},
		},

		{
			name:        "should return nil if fallback name clashes with existing portName",
			candidate:   corev1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"web": true, "port-8090": true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := filterPort(logger, test.candidate, test.portNumbers, test.portNames)
			if test.expected != (corev1.ServicePort{}) {
				assert.Equal(t, test.expected, *actual)
				return
			}
			assert.Nil(t, actual)

		})

	}
}

func TestDesiredCollectorService(t *testing.T) {
	t.Run("should return nil service for unknown receiver and protocol", func(t *testing.T) {
		params := Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{Config: `receivers:
      test:
        protocols:
          unknown:`},
			},
		}

		actual := desiredCollectorService(context.Background(), params)
		assert.Nil(t, actual)

	})
	t.Run("should return service with port mentioned in Instance.Spec.Ports and inferred ports", func(t *testing.T) {

		jaegerPorts := corev1.ServicePort{
			Name:     "jaeger-grpc",
			Protocol: "TCP",
			Port:     14250,
		}
		ports := append(paramsCollector().Instance.Spec.Ports, jaegerPorts)
		expected := serviceCollector("test-collector", ports)
		actual := desiredCollectorService(context.Background(), paramsCollector())

		assert.Equal(t, expected, *actual)

	})

}

func TestExpectedCollectorServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), paramsCollector(), []corev1.Service{serviceCollector("test-collector", paramsCollector().Instance.Spec.Ports)})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update service", func(t *testing.T) {
		serviceInstance := serviceCollector("test-collector", paramsCollector().Instance.Spec.Ports)
		createObjectIfNotExists(t, "test-collector", &serviceInstance)

		extraPorts := corev1.ServicePort{
			Name:       "port-web",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt(8080),
		}

		ports := append(paramsCollector().Instance.Spec.Ports, extraPorts)
		err := expectedServices(context.Background(), paramsCollector(), []corev1.Service{serviceCollector("test-collector", ports)})
		assert.NoError(t, err)

		actual := corev1.Service{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Contains(t, actual.Spec.Ports, extraPorts)

	})
}

func TestDeleteCollectorServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		ports := []corev1.ServicePort{{
			Port: 80,
			Name: "web",
		}}
		deleteService := serviceCollector("delete-service-collector", ports)
		createObjectIfNotExists(t, "delete-service-collector", &deleteService)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.True(t, exists)

		desired := desiredCollectorService(context.Background(), paramsCollector())
		opts := []client.ListOption{
			client.InNamespace(paramsTA().Instance.Namespace),
			client.MatchingLabels(map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", paramsCollector().Instance.Namespace, paramsCollector().Instance.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			}),
		}
		err = deleteServices(context.Background(), paramsCollector(), []corev1.Service{*desired}, opts)
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "delete-service-collector"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func TestHeadlessService(t *testing.T) {
	t.Run("should return headless service", func(t *testing.T) {
		actual := headless(context.Background(), paramsCollector())
		assert.Equal(t, actual.Spec.ClusterIP, "None")
	})
}

func TestMonitoringService(t *testing.T) {
	t.Run("returned service should expose monitoring port", func(t *testing.T) {
		expected := []corev1.ServicePort{{
			Name: "monitoring",
			Port: 8888,
		}}
		actual := monitoringService(context.Background(), paramsCollector())
		assert.Equal(t, expected, actual.Spec.Ports)

	})
}

func TestDesiredTAService(t *testing.T) {
	t.Run("should return service with default port", func(t *testing.T) {
		expected := serviceTA("test-targetallocator")
		actual := desiredTAService(paramsTA())

		assert.Equal(t, expected, actual)
	})

}

func TestExpectedTAServices(t *testing.T) {
	t.Run("should create the service", func(t *testing.T) {
		err := expectedServices(context.Background(), paramsTA(), []corev1.Service{serviceTA("targetallocator")})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
}

func TestDeleteTAServices(t *testing.T) {
	t.Run("should delete excess services", func(t *testing.T) {
		deleteService := serviceTA("test-delete-targetallocator", 8888)
		createObjectIfNotExists(t, "test-delete-targetallocator", &deleteService)

		exists, err := populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.True(t, exists)

		opts := []client.ListOption{
			client.InNamespace(paramsTA().Instance.Namespace),
			client.MatchingLabels(map[string]string{
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", paramsTA().Instance.Name, "targetallocator"),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			}),
		}
		err = deleteServices(context.Background(), paramsTA(), []corev1.Service{desiredTAService(paramsTA())}, opts)
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &corev1.Service{}, types.NamespacedName{Namespace: "default", Name: "test-delete-targetallocator"})
		assert.NoError(t, err)
		assert.False(t, exists)

	})
}

func serviceCollector(name string, ports []corev1.ServicePort) corev1.Service {
	labels := collector.Labels(paramsCollector().Instance)
	labels["app.kubernetes.io/name"] = name

	selector := labels
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: paramsCollector().Instance.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports:     ports,
		},
	}
}

func serviceTA(name string, portOpt ...int32) corev1.Service {
	port := int32(443)
	if len(portOpt) > 0 {
		port = portOpt[0]
	}
	params := paramsTA()
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
