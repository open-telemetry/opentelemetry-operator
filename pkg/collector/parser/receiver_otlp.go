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
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ ReceiverParser = &OTLPReceiverParser{}

const (
	parserNameOTLP = "__otlp"

	defaultOTLPGRPCPort       int32 = 4317
	defaultOTLPHTTPLegacyPort int32 = 55681
	defaultOTLPHTTPPort       int32 = 4318
)

// OTLPReceiverParser parses the configuration for OTLP receivers.
type OTLPReceiverParser struct {
	logger logr.Logger
	name   string
	config map[interface{}]interface{}
}

// NewOTLPReceiverParser builds a new parser for OTLP receivers.
func NewOTLPReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	if protocols, ok := config["protocols"].(map[interface{}]interface{}); ok {
		return &OTLPReceiverParser{
			logger: logger,
			name:   name,
			config: protocols,
		}
	}

	return &OTLPReceiverParser{
		name:   name,
		config: map[interface{}]interface{}{},
	}
}

// Ports returns all the service ports for all protocols in this parser.
func (o *OTLPReceiverParser) Ports() ([]corev1.ServicePort, error) {
	ports := []corev1.ServicePort{}

	for _, protocol := range []struct {
		name         string
		defaultPorts []corev1.ServicePort
	}{
		{
			name: "grpc",
			defaultPorts: []corev1.ServicePort{
				{
					Name:       portName(fmt.Sprintf("%s-grpc", o.name), defaultOTLPGRPCPort),
					Port:       defaultOTLPGRPCPort,
					TargetPort: intstr.FromInt(int(defaultOTLPGRPCPort)),
				},
			},
		},
		{
			name: "http",
			defaultPorts: []corev1.ServicePort{
				{
					Name:       portName(fmt.Sprintf("%s-http", o.name), defaultOTLPHTTPPort),
					Port:       defaultOTLPHTTPPort,
					TargetPort: intstr.FromInt(int(defaultOTLPHTTPPort)),
				},
				{
					Name:       portName(fmt.Sprintf("%s-http-legacy", o.name), defaultOTLPHTTPLegacyPort),
					Port:       defaultOTLPHTTPLegacyPort,
					TargetPort: intstr.FromInt(int(defaultOTLPHTTPPort)), // we target the official port, not the legacy
				},
			},
		},
	} {
		// do we have the protocol specified at all?
		if receiverProtocol, ok := o.config[protocol.name]; ok {
			// we have the specified protocol, we definitely need a service port
			nameWithProtocol := fmt.Sprintf("%s-%s", o.name, protocol.name)
			var protocolPort *corev1.ServicePort

			// do we have a configuration block for the protocol?
			settings, ok := receiverProtocol.(map[interface{}]interface{})
			if ok {
				protocolPort = singlePortFromConfigEndpoint(o.logger, nameWithProtocol, settings)
			}

			// have we parsed a port based on the configuration block?
			// if not, we use the default port
			if protocolPort == nil {
				ports = append(ports, protocol.defaultPorts...)
			} else {
				ports = append(ports, *protocolPort)
			}
		}
	}

	return ports, nil
}

// ParserName returns the name of this parser.
func (o *OTLPReceiverParser) ParserName() string {
	return parserNameOTLP
}

func init() {
	Register("otlp", NewOTLPReceiverParser)
}
