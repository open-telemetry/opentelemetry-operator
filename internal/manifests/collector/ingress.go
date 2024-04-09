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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func Ingress(params manifests.Params) (*networkingv1.Ingress, error) {
	name := naming.Ingress(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	if params.OtelCol.Spec.Ingress.Type != v1beta1.IngressTypeNginx {
		return nil, nil
	}

	ports, err := servicePortsFromCfg(params.Log, params.OtelCol)

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 || err != nil {
		params.Log.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", params.OtelCol.Name,
			"instance.namespace", params.OtelCol.Namespace,
		)
		return nil, err
	}

	var rules []networkingv1.IngressRule
	switch params.OtelCol.Spec.Ingress.RuleType {
	case v1beta1.IngressRuleTypePath, "":
		rules = []networkingv1.IngressRule{createPathIngressRules(params.OtelCol.Name, params.OtelCol.Spec.Ingress.Hostname, ports)}
	case v1beta1.IngressRuleTypeSubdomain:
		rules = createSubdomainIngressRules(params.OtelCol.Name, params.OtelCol.Spec.Ingress.Hostname, ports)
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Ingress(params.OtelCol.Name),
			Namespace:   params.OtelCol.Namespace,
			Annotations: params.OtelCol.Spec.Ingress.Annotations,
			Labels:      labels,
		},
		Spec: networkingv1.IngressSpec{
			TLS:              params.OtelCol.Spec.Ingress.TLS,
			Rules:            rules,
			IngressClassName: params.OtelCol.Spec.Ingress.IngressClassName,
		},
	}, nil
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

func servicePortsFromCfg(logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector) ([]corev1.ServicePort, error) {
	out, err := otelcol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}
	configFromString, err := adapters.ConfigFromString(out)
	if err != nil {
		logger.Error(err, "couldn't extract the configuration from the context")
		return nil, err
	}

	ports, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeReceiver, configFromString)
	if err != nil {
		logger.Error(err, "couldn't build the ingress for this instance")
		return nil, err
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
		var resultingInferredPorts []v1beta1.PortsSpec
		for _, inferred := range ports {
			if filtered := filterPort(logger, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}
		ports = append(otelcol.Spec.Ports, resultingInferredPorts...)
	}

	svcPorts := []corev1.ServicePort{}
	for _, p := range ports {
		svcPorts = append(svcPorts, p.ServicePort)
	}

	return svcPorts, err
}
