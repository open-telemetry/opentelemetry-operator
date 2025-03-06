// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// headless and monitoring labels are to differentiate the base/headless/monitoring services from the clusterIP service.
const (
	headlessLabel    = "operator.opentelemetry.io/collector-headless-service"
	monitoringLabel  = "operator.opentelemetry.io/collector-monitoring-service"
	serviceTypeLabel = "operator.opentelemetry.io/collector-service-type"
	valueExists      = "Exists"
)

type ServiceType int

const (
	BaseServiceType ServiceType = iota
	HeadlessServiceType
	MonitoringServiceType
	ExtensionServiceType
)

func (s ServiceType) String() string {
	return [...]string{"base", "headless", "monitoring", "extension"}[s]
}

func HeadlessService(params manifests.Params) (*corev1.Service, error) {
	h, err := Service(params)
	if h == nil || err != nil {
		return h, err
	}

	h.Name = naming.HeadlessService(params.OtelCol.Name)
	h.Labels[headlessLabel] = valueExists
	h.Labels[serviceTypeLabel] = HeadlessServiceType.String()

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
	name := naming.MonitoringService(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	labels[monitoringLabel] = valueExists
	labels[serviceTypeLabel] = MonitoringServiceType.String()

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	_, metricsPort, err := params.OtelCol.Spec.Config.Service.MetricsEndpoint(params.Log)
	if err != nil {
		return nil, err
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP: "",
			Ports: []corev1.ServicePort{{
				Name: "monitoring",
				Port: metricsPort,
			}},
			IPFamilies:     params.OtelCol.Spec.IpFamilies,
			IPFamilyPolicy: params.OtelCol.Spec.IpFamilyPolicy,
		},
	}, nil
}

func ExtensionService(params manifests.Params) (*corev1.Service, error) {
	name := naming.ExtensionService(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	labels[serviceTypeLabel] = ExtensionServiceType.String()

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	ports, err := params.OtelCol.Spec.Config.GetExtensionPorts(params.Log)
	if err != nil {
		return nil, err
	}

	if len(ports) == 0 {
		return nil, nil
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
		},
	}, nil
}

func Service(params manifests.Params) (*corev1.Service, error) {
	name := naming.Service(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	labels[serviceTypeLabel] = BaseServiceType.String()

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	ports, err := params.OtelCol.Spec.Config.GetReceiverAndExporterPorts(params.Log)
	if err != nil {
		return nil, err
	}

	// set appProtocol to h2c for grpc ports on OpenShift.
	// OpenShift uses HA proxy that uses appProtocol for its configuration.
	for i := range ports {
		h2c := "h2c"
		if params.OtelCol.Spec.Ingress.Type == v1beta1.IngressTypeRoute && ports[i].AppProtocol != nil && strings.EqualFold(*ports[i].AppProtocol, "grpc") {
			ports[i].AppProtocol = &h2c
		}
	}

	if len(params.OtelCol.Spec.Ports) > 0 {
		// we should add all the ports from the CR
		// there are two cases where problems might occur:
		// 1) when the port number is already being used by a receiver
		// 2) same, but for the port name
		//
		// in the first case, we remove the port we inferred from the list
		// in the second case, we rename our inferred port to something like "port-%d"
		portNumbers, portNames := extractPortNumbersAndNames(params.OtelCol.Spec.Ports)
		var resultingInferredPorts []corev1.ServicePort
		for _, inferred := range ports {
			if filtered := filterPort(params.Log, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}

		ports = append(toServicePorts(params.OtelCol.Spec.Ports), resultingInferredPorts...)
	}

	// if we have no ports, we don't need a service
	if len(ports) == 0 {

		params.Log.V(1).Info("the instance's configuration didn't yield any ports to open, skipping service", "instance.name", params.OtelCol.Name, "instance.namespace", params.OtelCol.Namespace)
		return nil, err
	}

	trafficPolicy := corev1.ServiceInternalTrafficPolicyCluster
	if params.OtelCol.Spec.Mode == v1beta1.ModeDaemonSet {
		trafficPolicy = corev1.ServiceInternalTrafficPolicyLocal
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Service(params.OtelCol.Name),
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			InternalTrafficPolicy: &trafficPolicy,
			Selector:              manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			ClusterIP:             "",
			Ports:                 ports,
			IPFamilies:            params.OtelCol.Spec.IpFamilies,
			IPFamilyPolicy:        params.OtelCol.Spec.IpFamilyPolicy,
		},
	}, nil
}

type PortNumberKey struct {
	Port     int32
	Protocol corev1.Protocol
}

func newPortNumberKeyByPort(port int32) PortNumberKey {
	return PortNumberKey{Port: port, Protocol: corev1.ProtocolTCP}
}

func newPortNumberKey(port int32, protocol corev1.Protocol) PortNumberKey {
	if protocol == "" {
		// K8s defaults to TCP if protocol is empty, so evaluate the port the same
		protocol = corev1.ProtocolTCP
	}
	return PortNumberKey{Port: port, Protocol: protocol}
}

func filterPort(logger logr.Logger, candidate corev1.ServicePort, portNumbers map[PortNumberKey]bool, portNames map[string]bool) *corev1.ServicePort {
	if portNumbers[newPortNumberKey(candidate.Port, candidate.Protocol)] {
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

func extractPortNumbersAndNames(ports []v1beta1.PortsSpec) (map[PortNumberKey]bool, map[string]bool) {
	numbers := map[PortNumberKey]bool{}
	names := map[string]bool{}

	for _, port := range ports {
		numbers[newPortNumberKey(port.Port, port.Protocol)] = true
		names[port.Name] = true
	}

	return numbers, names
}
