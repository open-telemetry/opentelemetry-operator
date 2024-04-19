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

package parser

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var _ PortRetriever = &SingleEndpointConfig{}

type SinglePortOption func(parser *SinglePortParser)

type SingleEndpointConfig struct {
	Endpoint      string `json:"endpoint,omitempty"`
	ListenAddress string `json:"listen_address,omitempty"`
}

func (g *SingleEndpointConfig) GetPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := g.GetPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default: %d", p)
		return p
	}
	return num
}

func (g *SingleEndpointConfig) GetPortNum() (int32, error) {
	if len(g.Endpoint) > 0 {
		return portFromEndpoint(g.Endpoint)
	} else if len(g.ListenAddress) > 0 {
		return portFromEndpoint(g.ListenAddress)
	}
	return 0, portNotFoundErr
}

func WithSinglePort(port int32, opts ...PortBuilderOption) SinglePortOption {
	return func(parser *SinglePortParser) {
		servicePort := &corev1.ServicePort{
			Name: naming.PortName(parser.name, port),
			Port: port,
		}
		for _, opt := range opts {
			opt(servicePort)
		}
		parser.svcPort = servicePort
	}
}

// SinglePortParser is a special parser for a generic receiver that has an endpoint or listen_address in its
// configuration. It doesn't self-register and should be created/used directly.
type SinglePortParser struct {
	config PortRetriever
	name   string

	svcPort *corev1.ServicePort
}

func CreateParser(opts ...SinglePortOption) Builder {
	return func(name string, config interface{}) (ComponentPortParser, error) {
		c := &SingleEndpointConfig{}
		if err := LoadMap[*SingleEndpointConfig](config, c); err != nil {
			return nil, err
		}
		r := &SinglePortParser{
			name:   name,
			config: c,
		}
		for _, opt := range opts {
			opt(r)
		}
		return r, nil
	}
}

func NewSinglePortParser(name string, config interface{}) (ComponentPortParser, error) {
	return CreateParser(WithSinglePort(0))(name, config)
}

func (g *SinglePortParser) ParserName() string {
	return fmt.Sprintf("__%s", ComponentType(g.name))
}

// Ports returns all the service ports for all protocols in this parser.
func (g *SinglePortParser) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	if _, err := g.config.GetPortNum(); err != nil && g.svcPort.Port == 0 {
		logger.WithValues("receiver", g.config).Error(err, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, err
	}

	port := g.svcPort.Port
	if g.config != nil {
		port = g.config.GetPortNumOrDefault(logger, port)
		g.svcPort.Name = naming.PortName(g.name, port)
	}
	return []corev1.ServicePort{ConstructServicePort(g.svcPort, port)}, nil
}
