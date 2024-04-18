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
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

type MultiPortOption func(parser *GenericMultiPortReceiver)
type PortBuilderOption func(portBuilder *corev1.ServicePort)

type GenericMultiPortReceiverConfig struct {
	Protocols map[string]endpointContainer `json:"protocols"`
}

// Receiver is a special parser for generic receivers. It doesn't self-register and should be created/used directly.
type GenericMultiPortReceiver struct {
	config *GenericMultiPortReceiverConfig
	name   string

	portMappings map[string]*corev1.ServicePort
}

func createMultiPortParser(opts ...MultiPortOption) parsers.Builder {
	return func(name string, config interface{}) (parsers.ComponentPortParser, error) {
		c := &GenericMultiPortReceiverConfig{}
		if err := parsers.LoadMap[GenericMultiPortReceiverConfig](config, c); err != nil {
			return nil, err
		}
		parser := &GenericMultiPortReceiver{
			name:         name,
			config:       c,
			portMappings: map[string]*corev1.ServicePort{},
		}
		for _, opt := range opts {
			opt(parser)
		}
		return parser, nil
	}
}

func NewGenericMultiPortReceiverParser(name string, config interface{}) (parsers.ComponentPortParser, error) {
	return createMultiPortParser()(name, config)
}

func (g *GenericMultiPortReceiver) ParserName() string {
	return fmt.Sprintf("__%s", g.ParserType())
}

func WithPortMapping(name string, port int32, opts ...PortBuilderOption) MultiPortOption {
	return func(parser *GenericMultiPortReceiver) {
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

func WithTargetPort(targetPort int32) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.TargetPort = intstr.FromInt32(targetPort)
	}
}
func WithNodePort(nodePort int32) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.NodePort = nodePort
	}
}

func WithAppProtocol(proto *string) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.AppProtocol = proto
	}
}

func WithProtocol(proto corev1.Protocol) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.Protocol = proto
	}
}

// ParserType retrieves the type for the receiver:
// - myreceiver/custom
// - myreceiver
// we extract the "myreceiver" part and see if we have a parser for the receiver
func (g *GenericMultiPortReceiver) ParserType() string {
	if strings.Contains(g.name, "/") {
		return g.name[:strings.Index(g.name, "/")]
	}

	return g.name
}

func (g *GenericMultiPortReceiver) IsScraper() bool {
	_, exists := scraperReceivers[g.ParserType()]
	return exists
}

// Ports returns all the service ports for all protocols in this parser.
func (g *GenericMultiPortReceiver) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	for protocol, ec := range g.config.Protocols {
		if defaultPort, ok := g.portMappings[protocol]; ok {
			ports = append(ports, constructServicePort(logger, defaultPort, ec))
		} else {
			return nil, errors.New(fmt.Sprintf("unknown protocol set: %s", protocol))
		}
	}

	return ports, nil
}

func constructServicePort(logger logr.Logger, current *corev1.ServicePort, ec endpointContainer) corev1.ServicePort {
	return corev1.ServicePort{
		Name:        current.Name,
		Port:        ec.getPortNumOrDefault(logger, current.Port),
		TargetPort:  current.TargetPort,
		NodePort:    current.NodePort,
		AppProtocol: current.AppProtocol,
		Protocol:    current.Protocol,
	}
}
