// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestExtractPortNumbersAndNames(t *testing.T) {
	t.Run("should return extracted port names and numbers", func(t *testing.T) {
		ports := []v1beta1.PortsSpec{
			{ServicePort: v1.ServicePort{Name: "web", Port: 8080}},
			{ServicePort: v1.ServicePort{Name: "tcp", Port: 9200}},
			{ServicePort: v1.ServicePort{Name: "web-explicit", Port: 80, Protocol: v1.ProtocolTCP}},
			{ServicePort: v1.ServicePort{Name: "syslog-udp", Port: 514, Protocol: v1.ProtocolUDP}},
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
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{Config: v1beta1.Config{}},
			},
		}

		actual, err := Service(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})
	t.Run("should return service with port mentioned in OtelCol.Spec.Ports and inferred ports", func(t *testing.T) {

		grpc := "grpc"
		jaegerPorts := v1beta1.PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        "jaeger-grpc",
				Protocol:    "TCP",
				Port:        14250,
				AppProtocol: &grpc,
			}}
		params := deploymentParams()
		ports := append(params.OtelCol.Spec.Ports, jaegerPorts)
		expected := service("test-collector", ports)

		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)

	})

	t.Run("on OpenShift gRPC appProtocol should be h2c", func(t *testing.T) {
		h2c := "h2c"
		jaegerPort := v1beta1.PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        "jaeger-grpc",
				Protocol:    "TCP",
				Port:        14250,
				AppProtocol: &h2c,
			}}

		params := deploymentParams()

		params.OtelCol.Spec.Ingress.Type = v1beta1.IngressTypeRoute
		actual, err := Service(params)

		ports := append(params.OtelCol.Spec.Ports, jaegerPort)
		expected := service("test-collector", ports)
		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)

	})

	t.Run("should return service with local internal traffic policy", func(t *testing.T) {

		grpc := "grpc"
		jaegerPorts := v1beta1.PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        "jaeger-grpc",
				Protocol:    "TCP",
				Port:        14250,
				AppProtocol: &grpc,
			}}
		p := paramsWithMode(v1beta1.ModeDaemonSet)
		ports := append(p.OtelCol.Spec.Ports, jaegerPorts)
		expected := serviceWithInternalTrafficPolicy("test-collector", ports, v1.ServiceInternalTrafficPolicyLocal)

		actual, err := Service(p)
		assert.NoError(t, err)

		assert.Equal(t, expected, *actual)
	})

	t.Run("should return service with OTLP ports", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{Config: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"otlp": map[string]interface{}{
								"protocols": map[string]interface{}{
									"grpc": nil,
									"http": nil,
								},
							},
						},
					},
					Exporters: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"otlp": map[string]interface{}{
								"endpoint": "jaeger-allinone-collector-headless.chainsaw-otlp-metrics.svc:4317",
							},
						},
					},
					Service: v1beta1.Service{
						Pipelines: map[string]*v1beta1.Pipeline{
							"traces": {
								Receivers: []string{"otlp"},
								Exporters: []string{"otlp"},
							},
						},
					},
				}},
			},
		}

		actual, err := Service(params)
		assert.NotNil(t, actual)
		assert.Len(t, actual.Spec.Ports, 2)
		assert.NoError(t, err)
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
		params.OtelCol.Spec.Config = v1beta1.Config{
			Service: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
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

func TestExtensionService(t *testing.T) {
	testCases := []struct {
		name          string
		params        manifests.Params
		expectedPorts []v1.ServicePort
	}{
		{
			name: "when the extension has http endpoint",
			params: manifests.Params{
				Config: config.Config{},
				Log:    logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Service: v1beta1.Service{
								Extensions: []string{"jaeger_query"},
							},
							Extensions: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"jaeger_query": map[string]interface{}{
										"http": map[string]interface{}{
											"endpoint": "0.0.0.0:16686",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedPorts: []v1.ServicePort{
				{
					Name: "jaeger-query",
					Port: 16686,
					TargetPort: intstr.IntOrString{
						IntVal: 16686,
					},
				},
			},
		},
		{
			name: "when the extension has grpc endpoint",
			params: manifests.Params{
				Config: config.Config{},
				Log:    logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Service: v1beta1.Service{
								Extensions: []string{"jaeger_query"},
							},
							Extensions: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"jaeger_query": map[string]interface{}{
										"http": map[string]interface{}{
											"endpoint": "0.0.0.0:16686",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedPorts: []v1.ServicePort{
				{
					Name: "jaeger-query",
					Port: 16686,
					TargetPort: intstr.IntOrString{
						IntVal: 16686,
					},
				},
			},
		},
		{
			name: "when the extension has both http and grpc endpoint",
			params: manifests.Params{
				Config: config.Config{},
				Log:    logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Service: v1beta1.Service{
								Extensions: []string{"jaeger_query"},
							},
							Extensions: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"jaeger_query": map[string]interface{}{
										"http": map[string]interface{}{
											"endpoint": "0.0.0.0:16686",
										},
										"grpc": map[string]interface{}{
											"endpoint": "0.0.0.0:16686",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedPorts: []v1.ServicePort{
				{
					Name: "jaeger-query",
					Port: 16686,
					TargetPort: intstr.IntOrString{
						IntVal: 16686,
					},
				},
			},
		},
		{
			name: "when the extension has no extensions defined",
			params: manifests.Params{
				Config: config.Config{},
				Log:    logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Service: v1beta1.Service{
								Extensions: []string{"jaeger_query"},
							},
							Extensions: &v1beta1.AnyConfig{
								Object: map[string]interface{}{},
							},
						},
					},
				},
			},
			expectedPorts: []v1.ServicePort{},
		},
		{
			name: "when the extension has no endpoint defined",
			params: manifests.Params{
				Config: config.Config{},
				Log:    logger,
				OtelCol: v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Config: v1beta1.Config{
							Service: v1beta1.Service{
								Extensions: []string{"jaeger_query"},
							},
							Extensions: &v1beta1.AnyConfig{
								Object: map[string]interface{}{
									"jaeger_query": map[string]interface{}{},
								},
							},
						},
					},
				},
			},
			expectedPorts: []v1.ServicePort{
				{
					Name: "jaeger-query",
					Port: 16686,
					TargetPort: intstr.IntOrString{
						IntVal: 16686,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ExtensionService(tc.params)
			assert.NoError(t, err)

			if len(tc.expectedPorts) > 0 {
				assert.NotNil(t, actual)
				assert.Equal(t, actual.Name, naming.ExtensionService(tc.params.OtelCol.Name))
				// ports assertion
				assert.Equal(t, len(tc.expectedPorts), len(actual.Spec.Ports))
				assert.Equal(t, tc.expectedPorts[0].Name, actual.Spec.Ports[0].Name)
				assert.Equal(t, tc.expectedPorts[0].Port, actual.Spec.Ports[0].Port)
				assert.Equal(t, tc.expectedPorts[0].TargetPort.IntVal, actual.Spec.Ports[0].TargetPort.IntVal)
			} else {
				// no ports, no service
				assert.Nil(t, actual)
			}
		})
	}
}

func service(name string, ports []v1beta1.PortsSpec) v1.Service {
	return serviceWithInternalTrafficPolicy(name, ports, v1.ServiceInternalTrafficPolicyCluster)
}

func serviceWithInternalTrafficPolicy(name string, ports []v1beta1.PortsSpec, internalTrafficPolicy v1.ServiceInternalTrafficPolicyType) v1.Service {
	params := deploymentParams()
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	labels[serviceTypeLabel] = BaseServiceType.String()

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return v1.Service{}
	}

	svcPorts := []v1.ServicePort{}
	for _, p := range ports {
		p.ServicePort.TargetPort = intstr.FromInt32(p.Port)
		svcPorts = append(svcPorts, p.ServicePort)
	}

	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			InternalTrafficPolicy: &internalTrafficPolicy,
			Selector:              manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP:             "",
			Ports:                 svcPorts,
		},
	}
}

func TestServiceWithIpFamily(t *testing.T) {
	t.Run("should return IPFamilies for IPV4 and IPV6", func(t *testing.T) {
		params := deploymentParams()
		params.OtelCol.Spec.IpFamilies = []v1.IPFamily{
			"IPv4",
			"IPv6",
		}
		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, actual.Spec.IPFamilies, []v1.IPFamily{
			"IPv4",
			"IPv6",
		})
	})
	t.Run("should return IPPolicy SingleStack", func(t *testing.T) {
		params := deploymentParams()
		baseIpFamily := v1.IPFamilyPolicySingleStack
		params.OtelCol.Spec.IpFamilyPolicy = &baseIpFamily
		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, actual.Spec.IPFamilyPolicy, params.OtelCol.Spec.IpFamilyPolicy)
	})
	t.Run("should return IPPolicy PreferDualStack", func(t *testing.T) {
		params := deploymentParams()
		baseIpFamily := v1.IPFamilyPolicyPreferDualStack
		params.OtelCol.Spec.IpFamilyPolicy = &baseIpFamily
		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, actual.Spec.IPFamilyPolicy, params.OtelCol.Spec.IpFamilyPolicy)
	})
	t.Run("should return IPPolicy RequireDualStack ", func(t *testing.T) {
		params := deploymentParams()
		baseIpFamily := v1.IPFamilyPolicyRequireDualStack
		params.OtelCol.Spec.IpFamilyPolicy = &baseIpFamily
		actual, err := Service(params)
		assert.NoError(t, err)
		assert.Equal(t, actual.Spec.IPFamilyPolicy, params.OtelCol.Spec.IpFamilyPolicy)
	})
}
