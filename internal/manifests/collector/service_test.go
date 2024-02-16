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

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestExtractPortNumbersAndNames(t *testing.T) {
	t.Run("should return extracted port names and numbers", func(t *testing.T) {
		ports := []v1.ServicePort{
			{Name: "web", Port: 8080},
			{Name: "tcp", Port: 9200},
			{Name: "web-explicit", Port: 80, Protocol: v1.ProtocolTCP},
			{Name: "syslog-udp", Port: 514, Protocol: v1.ProtocolUDP},
		}
		expectedPortNames := map[string]bool{"web": true, "tcp": true, "web-explicit": true, "syslog-udp": true}
		expectedPortNumbers := map[PortNumberKey]bool{
			newPortNumberKey(8080, v1.ProtocolTCP): true,
			newPortNumberKey(9200, v1.ProtocolTCP): true,
			newPortNumberKey(80, v1.ProtocolTCP):   true,
			newPortNumberKey(514, v1.ProtocolUDP):  true,
		}

		actualPortNumbers, actualPortNames := extractPortNumbersAndNames(ports)
		assert.Equal(t, expectedPortNames, actualPortNames)
		assert.Equal(t, expectedPortNumbers, actualPortNumbers)

	})
}

func TestFilterPort(t *testing.T) {

	tests := []struct {
		name        string
		candidate   v1.ServicePort
		portNumbers map[PortNumberKey]bool
		portNames   map[string]bool
		expected    v1.ServicePort
	}{
		{
			name:      "should filter out duplicate port",
			candidate: v1.ServicePort{Name: "web", Port: 8080},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
		},

		{
			name:      "should filter out duplicate port, protocol specified (TCP)",
			candidate: v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolTCP},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
		},

		{
			name:      "should filter out duplicate port, protocol specified (UDP)",
			candidate: v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolUDP},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKey(8080, v1.ProtocolUDP): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
		},

		{
			name:      "should not filter unique port",
			candidate: v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
			expected:  v1.ServicePort{Name: "web", Port: 8090},
		},

		{
			name:      "should not filter same port with different protocols",
			candidate: v1.ServicePort{Name: "web", Port: 8080},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKey(8080, v1.ProtocolUDP): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
			expected:  v1.ServicePort{Name: "web", Port: 8080},
		},

		{
			name:      "should not filter same port with different protocols, candidate has specified port (TCP vs UDP)",
			candidate: v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolTCP},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKey(8080, v1.ProtocolUDP): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
			expected:  v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolTCP},
		},

		{
			name:      "should not filter same port with different protocols, candidate has specified port (UDP vs TCP)",
			candidate: v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolUDP},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"test": true, "metrics": true},
			expected:  v1.ServicePort{Name: "web", Port: 8080, Protocol: v1.ProtocolUDP},
		},

		{
			name:      "should change the duplicate portName",
			candidate: v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"web": true, "metrics": true},
			expected:  v1.ServicePort{Name: "port-8090", Port: 8090},
		},

		{
			name:      "should return nil if fallback name clashes with existing portName",
			candidate: v1.ServicePort{Name: "web", Port: 8090},
			portNumbers: map[PortNumberKey]bool{
				newPortNumberKeyByPort(8080): true, newPortNumberKeyByPort(9200): true},
			portNames: map[string]bool{"web": true, "port-8090": true},
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
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1alpha2.OpenTelemetryCollector{
				Spec: v1alpha2.OpenTelemetryCollectorSpec{Config: v1alpha2.Config{}},
			},
		}

		actual, err := Service(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})
	t.Run("should return service with port mentioned in OtelCol.Spec.Ports and inferred ports", func(t *testing.T) {

		grpc := "grpc"
		jaegerPorts := v1.ServicePort{
			Name:        "jaeger-grpc",
			Protocol:    "TCP",
			Port:        14250,
			AppProtocol: &grpc,
		}
		params := deploymentParams()
		ports := append(params.OtelCol.Spec.Ports, jaegerPorts)
		expected := service("test-collector", ports)

		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)

	})

	t.Run("on OpenShift gRPC appProtocol should be h2c", func(t *testing.T) {
		h2c := "h2c"
		jaegerPort := v1.ServicePort{
			Name:        "jaeger-grpc",
			Protocol:    "TCP",
			Port:        14250,
			AppProtocol: &h2c,
		}

		params := deploymentParams()

		params.OtelCol.Spec.Ingress.Type = v1alpha2.IngressTypeRoute
		actual, err := Service(params)

		ports := append(params.OtelCol.Spec.Ports, jaegerPort)
		expected := service("test-collector", ports)
		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)

	})

	t.Run("should return service with local internal traffic policy", func(t *testing.T) {

		grpc := "grpc"
		jaegerPorts := v1.ServicePort{
			Name:        "jaeger-grpc",
			Protocol:    "TCP",
			Port:        14250,
			AppProtocol: &grpc,
		}
		p := paramsWithMode(v1alpha2.ModeDaemonSet)
		ports := append(p.OtelCol.Spec.Ports, jaegerPorts)
		expected := serviceWithInternalTrafficPolicy("test-collector", ports, v1.ServiceInternalTrafficPolicyLocal)

		actual, err := Service(p)
		assert.NoError(t, err)

		assert.Equal(t, expected, *actual)
	})

}

func TestHeadlessService(t *testing.T) {
	t.Run("should return headless service", func(t *testing.T) {
		param := deploymentParams()
		actual, err := HeadlessService(param)
		assert.NoError(t, err)
		assert.Equal(t, actual.GetAnnotations()["service.beta.openshift.io/serving-cert-secret-name"], "test-collector-headless-tls")
		assert.Equal(t, actual.Spec.ClusterIP, "None")
	})
}

func TestMonitoringService(t *testing.T) {
	t.Run("returned service should expose monitoring port in the default port", func(t *testing.T) {
		expected := []v1.ServicePort{{
			Name: "monitoring",
			Port: 8888,
		}}
		param := deploymentParams()

		actual, err := MonitoringService(param)
		assert.NoError(t, err)

		assert.Equal(t, expected, actual.Spec.Ports)
	})

	t.Run("returned the service in a custom port", func(t *testing.T) {
		expected := []v1.ServicePort{{
			Name: "monitoring",
			Port: 9090,
		}}
		params := deploymentParams()
		params.OtelCol.Spec.Config = v1alpha2.Config{
			Service: v1alpha2.Service{
				Telemetry: &v1alpha2.AnyConfig{
					Object: map[string]interface{}{
						"metrics": map[string]interface{}{
							"level":   "detailed",
							"address": "0.0.0.0:9090",
						},
					},
				},
			},
		}

		actual, err := MonitoringService(params)
		assert.NoError(t, err)

		assert.NotNil(t, actual)
		assert.Equal(t, expected, actual.Spec.Ports)
	})
}

func service(name string, ports []v1.ServicePort) v1.Service {
	return serviceWithInternalTrafficPolicy(name, ports, v1.ServiceInternalTrafficPolicyCluster)
}

func serviceWithInternalTrafficPolicy(name string, ports []v1.ServicePort, internalTrafficPolicy v1.ServiceInternalTrafficPolicyType) v1.Service {
	params := deploymentParams()
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})

	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: params.OtelCol.Annotations,
		},
		Spec: v1.ServiceSpec{
			InternalTrafficPolicy: &internalTrafficPolicy,
			Selector:              manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP:             "",
			Ports:                 ports,
		},
	}
}
