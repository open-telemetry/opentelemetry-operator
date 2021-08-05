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

package reconcilers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/reconcile"
)

func TestExtractPortNumbersAndNames(t *testing.T) {
	t.Run("should return extracted port names and numbers", func(t *testing.T) {
		ports := []v1.ServicePort{{Name: "web", Port: 8080}, {Name: "tcp", Port: 9200}}
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
		candidate   v1.ServicePort
		portNumbers map[int32]bool
		portNames   map[string]bool
		expected    v1.ServicePort
	}{
		{
			name:        "should filter out duplicate port",
			candidate:   v1.ServicePort{Name: "web", Port: 8080},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"test": true, "metrics": true},
		},

		{
			name:        "should not filter unique port",
			candidate:   v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"test": true, "metrics": true},
			expected:    v1.ServicePort{Name: "web", Port: 8090},
		},

		{
			name:        "should change the duplicate portName",
			candidate:   v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"web": true, "metrics": true},
			expected:    v1.ServicePort{Name: "port-8090", Port: 8090},
		},

		{
			name:        "should return nil if fallback name clashes with existing portName",
			candidate:   v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[int32]bool{8080: true, 9200: true},
			portNames:   map[string]bool{"web": true, "port-8090": true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := filterPort(logger, test.candidate, test.portNumbers, test.portNames)
			if test.expected != (v1.ServicePort{}) {
				assert.Equal(t, test.expected, *actual)
				return
			}
			assert.Nil(t, actual)

		})

	}
}

func TestDesiredService(t *testing.T) {
	t.Run("should return nil service for unknown receiver and protocol", func(t *testing.T) {
		params := reconcile.Params{
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

		actual := desiredService(context.Background(), params)
		assert.Nil(t, actual)

	})
	t.Run("should return service with port mentioned in Instance.Spec.Ports and inferred ports", func(t *testing.T) {

		jaegerPorts := v1.ServicePort{
			Name:     "jaeger-grpc",
			Protocol: "TCP",
			Port:     14250,
		}
		ports := append(params().Instance.Spec.Ports, jaegerPorts)
		expected := service("test-collector", ports)
		actual := desiredService(context.Background(), params())

		assert.Equal(t, expected, *actual)

	})

}

func TestHeadlessService(t *testing.T) {
	t.Run("should return headless service", func(t *testing.T) {
		actual := headless(context.Background(), params())
		assert.Equal(t, actual.Spec.ClusterIP, "None")
	})
}

func TestMonitoringService(t *testing.T) {
	t.Run("returned service should expose monitoring port", func(t *testing.T) {
		expected := []v1.ServicePort{{
			Name: "monitoring",
			Port: 8888,
		}}
		actual := monitoringService(context.Background(), params())
		assert.Equal(t, expected, actual.Spec.Ports)

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
