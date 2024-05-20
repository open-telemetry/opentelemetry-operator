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
	_                     components.ComponentPortParser = &SingleEndpointParser{}
	grpc                                                 = "grpc"
	http                                                 = "http"
	unsetPort             int32                          = 0
	singleEndpointConfigs                                = []components.ComponentPortParser{
		NewSinglePortParser("awsxray", 2000),
		NewSinglePortParser("carbon", 2003),
		NewSinglePortParser("collectd", 8081),
		NewSinglePortParser("fluentforward", 8006),
		NewSinglePortParser("influxdb", 8086),
		NewSinglePortParser("opencensus", 55678, components.WithAppProtocol(nil)),
		NewSinglePortParser("sapm", 7276),
		NewSinglePortParser("signalfx", 9943),
		NewSinglePortParser("splunk_hec", 8088),
		NewSinglePortParser("statsd", 8125, components.WithProtocol(corev1.ProtocolUDP)),
		NewSinglePortParser("tcplog", unsetPort, components.WithProtocol(corev1.ProtocolTCP)),
		NewSinglePortParser("udplog", unsetPort, components.WithProtocol(corev1.ProtocolUDP)),
		NewSinglePortParser("wavefront", 2003),
		NewSinglePortParser("zipkin", 9411, components.WithAppProtocol(&http), components.WithProtocol(corev1.ProtocolTCP)),
	}
)

// SingleEndpointConfig represents the minimal struct for a given YAML configuration input containing either
// endpoint or listen_address.
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
		return components.PortFromEndpoint(g.Endpoint)
	} else if len(g.ListenAddress) > 0 {
		return components.PortFromEndpoint(g.ListenAddress)
	}
	return 0, components.PortNotFoundErr
}

// SingleEndpointParser is a special parser for a generic receiver that has an endpoint or listen_address in its
// configuration. It doesn't self-register and should be created/used directly.
type SingleEndpointParser struct {
	name string

	svcPort *corev1.ServicePort
}

func (s *SingleEndpointParser) Ports(logger logr.Logger, config interface{}) ([]corev1.ServicePort, error) {
	singleEndpointConfig := &SingleEndpointConfig{}
	if err := components.LoadMap[*SingleEndpointConfig](config, singleEndpointConfig); err != nil {
		return nil, err
	}
	if _, err := singleEndpointConfig.GetPortNum(); err != nil && s.svcPort.Port == unsetPort {
		logger.WithValues("receiver", s.name).Error(err, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, err
	}

	port := singleEndpointConfig.GetPortNumOrDefault(logger, s.svcPort.Port)
	s.svcPort.Name = naming.PortName(s.name, port)
	return []corev1.ServicePort{components.ConstructServicePort(s.svcPort, port)}, nil
}

func (s *SingleEndpointParser) ParserType() string {
	return components.ComponentType(s.name)
}

func (s *SingleEndpointParser) ParserName() string {
	return fmt.Sprintf("__%s", s.name)
}

func NewSinglePortParser(name string, port int32, opts ...components.PortBuilderOption) *SingleEndpointParser {
	servicePort := &corev1.ServicePort{
		Name: naming.PortName(name, port),
		Port: port,
	}
	for _, opt := range opts {
		opt(servicePort)
	}
	return &SingleEndpointParser{name: name, svcPort: servicePort}
}
