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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func Ingress(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *networkingv1.Ingress {
	if otelcol.Spec.Ingress.Type != v1alpha1.IngressTypeNginx {
		return nil
	}

	ports := servicePortsFromCfg(logger, otelcol)

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 {
		logger.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", otelcol.Name,
			"instance.namespace", otelcol.Namespace,
		)
		return nil
	}

	var rules []networkingv1.IngressRule
	switch otelcol.Spec.Ingress.RuleType {
	case v1alpha1.IngressRuleTypePath, "":
		rules = []networkingv1.IngressRule{createPathIngressRules(otelcol.Name, otelcol.Spec.Ingress.Hostname, ports)}
	case v1alpha1.IngressRuleTypeSubdomain:
		rules = createSubdomainIngressRules(otelcol.Name, otelcol.Spec.Ingress.Hostname, ports)
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Ingress(otelcol.Name),
			Namespace:   otelcol.Namespace,
			Annotations: otelcol.Spec.Ingress.Annotations,
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.Ingress(otelcol.Name),
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: networkingv1.IngressSpec{
			TLS:              otelcol.Spec.Ingress.TLS,
			Rules:            rules,
			IngressClassName: otelcol.Spec.Ingress.IngressClassName,
		},
	}
}

func createPathIngressRules(otelcol string, hostname string, ports []corev1.ServicePort) networkingv1.IngressRule {
	pathType := networkingv1.PathTypePrefix
	paths := make([]networkingv1.HTTPIngressPath, len(ports))
	for i, port := range ports {
		portName := naming.PortName(port.Name, port.Port)
		paths[i] = networkingv1.HTTPIngressPath{
			Path:     "/" + port.Name,
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: naming.Service(otelcol),
					Port: networkingv1.ServiceBackendPort{
						Name: portName,
					},
				},
			},
		}
	}
	return networkingv1.IngressRule{
		Host: hostname,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: paths,
			},
		},
	}
}

func createSubdomainIngressRules(otelcol string, hostname string, ports []corev1.ServicePort) []networkingv1.IngressRule {
	var rules []networkingv1.IngressRule
	pathType := networkingv1.PathTypePrefix
	for _, port := range ports {
		portName := naming.PortName(port.Name, port.Port)

		host := fmt.Sprintf("%s.%s", portName, hostname)
		// This should not happen due to validation in the webhook.
		if hostname == "" || hostname == "*" {
			host = portName
		}
		rules = append(rules, networkingv1.IngressRule{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: naming.Service(otelcol),
									Port: networkingv1.ServiceBackendPort{
										Name: portName,
									},
								},
							},
						},
					},
				},
			},
		})
	}
	return rules
}

// TODO: Update this to properly return an error https://github.com/open-telemetry/opentelemetry-operator/issues/1972
func servicePortsFromCfg(logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []corev1.ServicePort {
	configFromString, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		logger.Error(err, "couldn't extract the configuration from the context")
		return nil
	}

	ports, err := adapters.ConfigToReceiverPorts(logger, configFromString)
	if err != nil {
		logger.Error(err, "couldn't build the ingress for this instance")
	}

	if len(otelcol.Spec.Ports) > 0 {
		// we should add all the ports from the CR
		// there are two cases where problems might occur:
		// 1) when the port number is already being used by a receiver
		// 2) same, but for the port name
		//
		// in the first case, we remove the port we inferred from the list
		// in the second case, we rename our inferred port to something like "port-%d"
		portNumbers, portNames := extractPortNumbersAndNames(otelcol.Spec.Ports)
		var resultingInferredPorts []corev1.ServicePort
		for _, inferred := range ports {
			if filtered := filterPort(logger, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}

		ports = append(otelcol.Spec.Ports, resultingInferredPorts...)
	}
	return ports
}
