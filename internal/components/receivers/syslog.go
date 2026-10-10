// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

type syslogReceiverConfig struct {
	TCP *components.SingleEndpointConfig `mapstructure:"tcp"`
	UDP *components.SingleEndpointConfig `mapstructure:"udp"`
}

func newSyslogParser() *components.GenericParser[*syslogReceiverConfig] {
	return components.NewBuilder[*syslogReceiverConfig]().
		WithName("syslog").
		WithPort(components.UnsetPort).
		WithPortParser(parseSyslogPort).
		MustBuild()
}

func parseSyslogPort(
	_ logr.Logger,
	name string,
	defaultPort *corev1.ServicePort,
	config *syslogReceiverConfig,
) ([]corev1.ServicePort, error) {
	if config == nil {
		return nil, nil
	}

	endpoint := config.TCP
	protocol := corev1.ProtocolTCP
	if endpoint == nil {
		endpoint = config.UDP
		protocol = corev1.ProtocolUDP
	}
	if endpoint == nil {
		return nil, nil
	}

	port, err := endpoint.GetPortNum()
	if err != nil {
		return nil, err
	}

	servicePort := *defaultPort
	servicePort.Name = naming.PortName(name, port)
	servicePort.Protocol = protocol
	return []corev1.ServicePort{components.ConstructServicePort(&servicePort, port)}, nil
}
