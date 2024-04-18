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

package receiver

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

type multiProtoConfig interface {
	configByProtocol() map[string]*parser.SingleEndpointConfig
}

type MultiPortOption func(parser *MultiPortReceiver)

type multiProtocolEndpointConfig struct {
	Protocols map[string]*parser.SingleEndpointConfig `json:"protocols"`
}

var _ multiProtoConfig = &multiProtocolEndpointConfig{}

func (m *multiProtocolEndpointConfig) configByProtocol() map[string]*parser.SingleEndpointConfig {
	return m.Protocols
}

// MultiPortReceiver is a special parser for generic receivers. It doesn't self-register and should be created/used directly.
type MultiPortReceiver struct {
	config multiProtoConfig
	name   string

	portMappings map[string]*corev1.ServicePort
}

func createMultiPortParser[T multiProtoConfig](base T, opts ...MultiPortOption) parser.Builder {
	return func(name string, config interface{}) (parser.ComponentPortParser, error) {
		if err := parser.LoadMap[T](config, base); err != nil {
			return nil, err
		}
		multiReceiver := &MultiPortReceiver{
			name:         name,
			config:       base,
			portMappings: map[string]*corev1.ServicePort{},
		}
		for _, opt := range opts {
			opt(multiReceiver)
		}
		return multiReceiver, nil
	}
}

func (g *MultiPortReceiver) ParserName() string {
	return fmt.Sprintf("__%s", parser.ComponentType(g.name))
}

func WithPortMapping(name string, port int32, opts ...parser.PortBuilderOption) MultiPortOption {
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

func (g *MultiPortReceiver) IsScraper() bool {
	_, exists := scraperReceivers[parser.ComponentType(g.name)]
	return exists
}

// Ports returns all the service ports for all protocols in this parser.
func (g *MultiPortReceiver) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	if g.IsScraper() {
		return nil, nil
	}
	var ports []corev1.ServicePort
	for protocol, ec := range g.config.configByProtocol() {
		if defaultSvc, ok := g.portMappings[protocol]; ok {
			port := defaultSvc.Port
			if ec != nil {
				port = ec.GetPortNumOrDefault(logger, port)
			}
			ports = append(ports, parser.ConstructServicePort(defaultSvc, port))
		} else {
			return nil, errors.New(fmt.Sprintf("unknown protocol set: %s", protocol))
		}
	}
	return ports, nil
}
