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

package receivers

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var (
	_                  components.ComponentPortParser = &MultiPortReceiver{}
	multiPortReceivers                                = []components.ComponentPortParser{
		NewMultiPortReceiver("otlp",
			WithPortMapping(
				"grpc",
				4317,
				components.WithAppProtocol(&components.grpc),
				components.WithTargetPort(4317),
			), WithPortMapping(
				"http",
				4318,
				components.WithAppProtocol(&components.http),
				components.WithTargetPort(4318),
			),
		),
		NewMultiPortReceiver("skywalking",
			WithPortMapping(components.grpc, 11800,
				components.WithTargetPort(11800),
				components.WithAppProtocol(&components.grpc),
			),
			WithPortMapping(components.http, 12800,
				components.WithTargetPort(12800),
				components.WithAppProtocol(&components.http),
			)),
		NewMultiPortReceiver("jaeger",
			WithPortMapping(components.grpc, 14250,
				components.WithProtocol(corev1.ProtocolTCP),
				components.WithAppProtocol(&components.grpc),
			),
			WithPortMapping("thrift_http", 14268,
				components.WithProtocol(corev1.ProtocolTCP),
				components.WithAppProtocol(&components.http),
			),
			WithPortMapping("thrift_compact", 6831,
				components.WithProtocol(corev1.ProtocolUDP),
			),
			WithPortMapping("thrift_binary", 6832,
				components.WithProtocol(corev1.ProtocolUDP),
			),
		),
		NewMultiPortReceiver("loki",
			WithPortMapping(components.grpc, 9095,
				components.WithTargetPort(9095),
				components.WithAppProtocol(&components.grpc),
			),
			WithPortMapping(components.http, 3100,
				components.WithTargetPort(3100),
				components.WithAppProtocol(&components.http),
			),
		),
	}
)

// MultiProtocolEndpointConfig represents the minimal struct for a given YAML configuration input containing a map to
// a struct with either endpoint or listen_address.
type MultiProtocolEndpointConfig struct {
	Protocols map[string]*SingleEndpointConfig `json:"protocols"`
}

// MultiPortOption allows the setting of options for a MultiPortReceiver.
type MultiPortOption func(parser *MultiPortReceiver)

// MultiPortReceiver is a special parser for components with endpoints for each protocol.
type MultiPortReceiver struct {
	name string

	portMappings map[string]*corev1.ServicePort
}

func (m *MultiPortReceiver) Ports(logger logr.Logger, config interface{}) ([]corev1.ServicePort, error) {
	multiProtoEndpointCfg := &MultiProtocolEndpointConfig{}
	if err := components.LoadMap[*MultiProtocolEndpointConfig](config, multiProtoEndpointCfg); err != nil {
		return nil, err
	}
	var ports []corev1.ServicePort
	for protocol, ec := range multiProtoEndpointCfg.Protocols {
		if defaultSvc, ok := m.portMappings[protocol]; ok {
			port := defaultSvc.Port
			if ec != nil {
				port = ec.GetPortNumOrDefault(logger, port)
				defaultSvc.Name = naming.PortName(fmt.Sprintf("%s-%s", m.name, protocol), port)
			}
			ports = append(ports, components.ConstructServicePort(defaultSvc, port))
		} else {
			return nil, fmt.Errorf("unknown protocol set: %s", protocol)
		}
	}
	return ports, nil
}

func (m *MultiPortReceiver) ParserType() string {
	return components.ComponentType(m.name)
}

func (m *MultiPortReceiver) ParserName() string {
	return fmt.Sprintf("__%s", m.name)
}

func NewMultiPortReceiver(name string, opts ...MultiPortOption) *MultiPortReceiver {
	multiReceiver := &MultiPortReceiver{
		name:         name,
		portMappings: map[string]*corev1.ServicePort{},
	}
	for _, opt := range opts {
		opt(multiReceiver)
	}
	return multiReceiver
}

func WithPortMapping(name string, port int32, opts ...components.PortBuilderOption) MultiPortOption {
	return func(parser *MultiPortReceiver) {
		servicePort := &corev1.ServicePort{
			Name: naming.PortName(fmt.Sprintf("%s-%s", parser.name, name), port),
			Port: port,
		}
		for _, opt := range opts {
			opt(servicePort)
		}
		parser.portMappings[name] = servicePort
	}
}
