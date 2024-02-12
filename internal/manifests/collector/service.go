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
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/api/convert"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// headless label is to differentiate the headless service from the clusterIP service.
const (
	headlessLabel  = "operator.opentelemetry.io/collector-headless-service"
	headlessExists = "Exists"
)

func HeadlessService(params manifests.Params) (*corev1.Service, error) {
	h, err := Service(params)
	if h == nil || err != nil {
		return h, err
	}

	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}

	h.Name = naming.HeadlessService(otelCol.Name)
	h.Labels[headlessLabel] = headlessExists

	// copy to avoid modifying params.OtelCol.Annotations
	annotations := map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", h.Name),
	}
	for k, v := range h.Annotations {
		annotations[k] = v
	}
	h.Annotations = annotations

	h.Spec.ClusterIP = "None"
	return h, nil
}

func MonitoringService(params manifests.Params) (*corev1.Service, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}

	name := naming.MonitoringService(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, name, otelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})

	out, err := otelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	c, err := adapters.ConfigFromString(out)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration")
		return nil, err
	}

	metricsPort, err := adapters.ConfigToMetricsPort(params.Log, c)
	if err != nil {
		return nil, err
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelCol.Namespace,
			Labels:      labels,
			Annotations: otelCol.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  manifestutils.SelectorLabels(otelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP: "",
			Ports: []corev1.ServicePort{{
				Name: "monitoring",
				Port: metricsPort,
			}},
		},
	}, nil
}

func Service(params manifests.Params) (*corev1.Service, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}
	name := naming.Service(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, name, otelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})

	out, err := otelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	configFromString, err := adapters.ConfigFromString(out)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil, err
	}

	ports, err := adapters.ConfigToPorts(params.Log, configFromString)
	if err != nil {
		return nil, err
	}

	// set appProtocol to h2c for grpc ports on OpenShift.
	// OpenShift uses HA proxy that uses appProtocol for its configuration.
	for i := range ports {
		h2c := "h2c"
		if otelCol.Spec.Ingress.Type == v1alpha2.IngressTypeRoute && ports[i].AppProtocol != nil && strings.EqualFold(*ports[i].AppProtocol, "grpc") {
			ports[i].AppProtocol = &h2c
		}
	}

	if len(otelCol.Spec.Ports) > 0 {
		// we should add all the ports from the CR
		// there are two cases where problems might occur:
		// 1) when the port number is already being used by a receiver
		// 2) same, but for the port name
		//
		// in the first case, we remove the port we inferred from the list
		// in the second case, we rename our inferred port to something like "port-%d"
		portNumbers, portNames := extractPortNumbersAndNames(otelCol.Spec.Ports)
		var resultingInferredPorts []corev1.ServicePort
		for _, inferred := range ports {
			if filtered := filterPort(params.Log, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}

		ports = append(otelCol.Spec.Ports, resultingInferredPorts...)
	}

	// if we have no ports, we don't need a service
	if len(ports) == 0 {

		params.Log.V(1).Info("the instance's configuration didn't yield any ports to open, skipping service", "instance.name", otelCol.Name, "instance.namespace", otelCol.Namespace)
		return nil, err
	}

	trafficPolicy := corev1.ServiceInternalTrafficPolicyCluster
	if otelCol.Spec.Mode == v1alpha2.ModeDaemonSet {
		trafficPolicy = corev1.ServiceInternalTrafficPolicyLocal
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Service(otelCol.Name),
			Namespace:   otelCol.Namespace,
			Labels:      labels,
			Annotations: otelCol.Annotations,
		},
		Spec: corev1.ServiceSpec{
			InternalTrafficPolicy: &trafficPolicy,
			Selector:              manifestutils.SelectorLabels(otelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP:             "",
			Ports:                 ports,
		},
	}, nil
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
