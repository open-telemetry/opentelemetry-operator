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

package components

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var (
	_ ComponentPortParser = &SingleEndpointParser{}
)

// SingleEndpointConfig represents the minimal struct for a given YAML configuration input containing either
// endpoint or listen_address.
type SingleEndpointConfig struct {
	Endpoint      string `mapstructure:"endpoint,omitempty"`
	ListenAddress string `mapstructure:"listen_address,omitempty"`
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
		return PortFromEndpoint(g.Endpoint)
	} else if len(g.ListenAddress) > 0 {
		return PortFromEndpoint(g.ListenAddress)
	}
	return 0, PortNotFoundErr
}

// SingleEndpointParser is a special parser for a generic receiver that has an endpoint or listen_address in its
// configuration. It doesn't self-register and should be created/used directly.
type SingleEndpointParser struct {
	name string

	svcPort *corev1.ServicePort
}

func (s *SingleEndpointParser) Ports(logger logr.Logger, config interface{}) ([]corev1.ServicePort, error) {
	singleEndpointConfig := &SingleEndpointConfig{}
	if err := mapstructure.Decode(config, singleEndpointConfig); err != nil {
		return nil, err
	}
	if _, err := singleEndpointConfig.GetPortNum(); err != nil && s.svcPort.Port == UnsetPort {
		logger.WithValues("receiver", s.name).Error(err, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, err
	}

	port := singleEndpointConfig.GetPortNumOrDefault(logger, s.svcPort.Port)
	s.svcPort.Name = naming.PortName(s.name, port)
	return []corev1.ServicePort{ConstructServicePort(s.svcPort, port)}, nil
}

func (s *SingleEndpointParser) ParserType() string {
	return ComponentType(s.name)
}

func (s *SingleEndpointParser) ParserName() string {
	return fmt.Sprintf("__%s", s.name)
}

func NewSinglePortParser(name string, port int32, opts ...PortBuilderOption) *SingleEndpointParser {
	servicePort := &corev1.ServicePort{
		Name: naming.PortName(name, port),
		Port: port,
	}
	for _, opt := range opts {
		opt(servicePort)
	}
	return &SingleEndpointParser{name: name, svcPort: servicePort}
}
