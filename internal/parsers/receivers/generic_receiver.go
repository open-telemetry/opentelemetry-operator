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

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

type Option func(parser *Receiver)

type GenericReceiverConfig struct {
	Endpoint      string `json:"endpoint,omitempty"`
	ListenAddress string `json:"listen_address,omitempty"`
}

func (g *GenericReceiverConfig) getPortNum() (int32, error) {
	if len(g.Endpoint) > 0 {
		return portFromEndpoint(g.Endpoint)
	} else if len(g.ListenAddress) > 0 {
		return portFromEndpoint(g.ListenAddress)
	}
	return 0, errors.New("port should not be empty")
}

// Receiver is a special parser for generic receivers. It doesn't self-register and should be created/used directly.
type Receiver struct {
	config *GenericReceiverConfig
	name   string

	// Optional fields
	defaultAppProtocol *string
	defaultProtocol    corev1.Protocol
	defaultPort        int32
}

func createParser(opts ...Option) parsers.Builder {
	return func(name string, config interface{}) (parsers.ComponentPortParser, error) {
		c := &GenericReceiverConfig{}
		if err := parsers.LoadMap[GenericReceiverConfig](config, c); err != nil {
			return nil, err
		}
		parser := &Receiver{
			name:   name,
			config: c,
		}
		for _, opt := range opts {
			opt(parser)
		}
		return parser, nil
	}
}

func NewGenericReceiverParser(name string, config interface{}) (parsers.ComponentPortParser, error) {
	return createParser()(name, config)
}

func (g *Receiver) ParserName() string {
	return fmt.Sprintf("__%s", g.ParserType())
}

func WithDefaultAppProtocol(proto *string) Option {
	return func(receiver *Receiver) {
		receiver.defaultAppProtocol = proto
	}
}

func WithDefaultProtocol(proto corev1.Protocol) Option {
	return func(receiver *Receiver) {
		receiver.defaultProtocol = proto
	}
}

func WithDefaultPort(port int32) Option {
	return func(receiver *Receiver) {
		receiver.defaultPort = port
	}
}

// ParserType retrieves the type for the receiver:
// - myreceiver/custom
// - myreceiver
// we extract the "myreceiver" part and see if we have a parser for the receiver
func (g *Receiver) ParserType() string {
	if strings.Contains(g.name, "/") {
		return g.name[:strings.Index(g.name, "/")]
	}

	return g.name
}

func (g *Receiver) IsScraper() bool {
	_, exists := scraperReceivers[g.ParserType()]
	return exists
}

// Ports returns all the service ports for all protocols in this parser.
func (g *Receiver) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	if g.IsScraper() {
		return nil, nil
	}
	portNum, err := g.config.getPortNum()
	if err != nil && g.defaultPort > 0 {
		logger.WithValues("receiver", g.config).Error(err, "couldn't parse the endpoint's port")
		return []corev1.ServicePort{{
			Port:        g.defaultPort,
			Name:        naming.PortName(g.name, g.defaultPort),
			Protocol:    g.defaultProtocol,
			AppProtocol: g.defaultAppProtocol,
		}}, nil
	} else if err != nil {
		logger.WithValues("receiver", g.config).Error(err, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, err
	}

	return []corev1.ServicePort{
		{
			Name:        naming.PortName(g.name, portNum),
			Port:        portNum,
			Protocol:    g.defaultProtocol,
			AppProtocol: g.defaultAppProtocol,
		},
	}, nil
}
