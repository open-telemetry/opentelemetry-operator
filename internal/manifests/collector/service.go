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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// headless label is to differentiate the headless service from the clusterIP service.
const (
	headlessLabel  = "operator.opentelemetry.io/collector-headless-service"
	headlessExists = "Exists"
)

func HeadlessService(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.Service {
	h := Service(cfg, logger, otelcol)
	if h == nil {
		return h
	}

	h.Name = naming.HeadlessService(otelcol.Name)
	h.Labels[headlessLabel] = headlessExists

	// copy to avoid modifying otelcol.Annotations
	annotations := map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", h.Name),
	}
	for k, v := range h.Annotations {
		annotations[k] = v
	}
	h.Annotations = annotations

	h.Spec.ClusterIP = "None"
	return h
}

func MonitoringService(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.Service {
	name := naming.MonitoringService(otelcol.Name)
	labels := Labels(otelcol, name, []string{})

	c, err := adapters.ConfigFromString(otelcol.Spec.Config)
	// TODO: Update this to properly return an error https://github.com/open-telemetry/opentelemetry-operator/issues/1972
	if err != nil {
		logger.Error(err, "couldn't extract the configuration")
		return nil
	}

	metricsPort, err := adapters.ConfigToMetricsPort(logger, c)
	if err != nil {
		logger.V(2).Info("couldn't determine metrics port from configuration, using 8888 default value", "error", err)
		metricsPort = 8888
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  SelectorLabels(otelcol),
			ClusterIP: "",
			Ports: []corev1.ServicePort{{
				Name: "monitoring",
				Port: metricsPort,
			}},
		},
	}
}

func Service(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.Service {
	name := naming.Service(otelcol.Name)
	labels := Labels(otelcol, name, []string{})

	configFromString, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		logger.Error(err, "couldn't extract the configuration from the context")
		return nil
	}

	ports, err := adapters.ConfigToReceiverPorts(logger, configFromString)
	if err != nil {
		logger.Error(err, "couldn't build the service for this instance")
		return nil
	}

	// set appProtocol to h2c for grpc ports on OpenShift.
	// OpenShift uses HA proxy that uses appProtocol for its configuration.
	for i, _ := range ports {
		h2c := "h2c"
		if otelcol.Spec.Ingress.Type == v1alpha1.IngressTypeRoute && ports[i].AppProtocol != nil && *(ports[i].AppProtocol) == "grpc" {
			ports[i].AppProtocol = &h2c
		}
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

	// if we have no ports, we don't need a service
	if len(ports) == 0 {
		logger.V(1).Info("the instance's configuration didn't yield any ports to open, skipping service", "instance.name", otelcol.Name, "instance.namespace", otelcol.Namespace)
		return nil
	}

	trafficPolicy := corev1.ServiceInternalTrafficPolicyCluster
	if otelcol.Spec.Mode == v1alpha1.ModeDaemonSet {
		trafficPolicy = corev1.ServiceInternalTrafficPolicyLocal
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Service(otelcol.Name),
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
		Spec: corev1.ServiceSpec{
			InternalTrafficPolicy: &trafficPolicy,
			Selector:              SelectorLabels(otelcol),
			ClusterIP:             "",
			Ports:                 ports,
		},
	}
}

func filterPort(logger logr.Logger, candidate corev1.ServicePort, portNumbers map[int32]bool, portNames map[string]bool) *corev1.ServicePort {
	if portNumbers[candidate.Port] {
		return nil
	}

	// do we have the port name there already?
	if portNames[candidate.Name] {
		// there's already a port with the same name! do we have a 'port-%d' already?
		fallbackName := fmt.Sprintf("port-%d", candidate.Port)
		if portNames[fallbackName] {
			// that wasn't expected, better skip this port
			logger.V(2).Info("a port name specified in the CR clashes with an inferred port name, and the fallback port name clashes with another port name! Skipping this port.",
				"inferred-port-name", candidate.Name,
				"fallback-port-name", fallbackName,
			)
			return nil
		}

		candidate.Name = fallbackName
		return &candidate
	}

	// this port is unique, return as is
	return &candidate
}

func extractPortNumbersAndNames(ports []corev1.ServicePort) (map[int32]bool, map[string]bool) {
	numbers := map[int32]bool{}
	names := map[string]bool{}

	for _, port := range ports {
		numbers[port.Port] = true
		names[port.Name] = true
	}

	return numbers, names
}
