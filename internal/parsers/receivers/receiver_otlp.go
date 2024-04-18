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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

var _ parsers.ComponentPortParser = &OTLPReceiverParser{}

const (
	parserNameOTLP = "__otlp"

	defaultOTLPGRPCPort int32 = 4317
	defaultOTLPHTTPPort int32 = 4318
)

var (
	grpc = "grpc"
	http = "http"
)

type Protocols struct {
	Grpc *endpointContainer `json:"grpc"`
	HTTP *endpointContainer `json:"http"`
}
type OTLPReceiverConfig struct {
	Protocols Protocols `json:"protocols"`
}

// OTLPReceiverParser parses the configuration for OTLP receivers.
type OTLPReceiverParser struct {
	config *OTLPReceiverConfig
	name   string
}

// NewOTLPReceiverParser builds a new parser for OTLP receivers.
func NewOTLPReceiverParser(name string, config interface{}) (parsers.ComponentPortParser, error) {
	c := &OTLPReceiverConfig{}
	if err := parsers.LoadMap[OTLPReceiverConfig](config, c); err != nil {
		return nil, err
	}
	return &OTLPReceiverParser{
		name:   name,
		config: c,
	}, nil
}

// Ports returns all the service ports for all protocols in this parser.
func (o *OTLPReceiverParser) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	if o.config.Protocols.Grpc != nil {
		ports = append(ports, corev1.ServicePort{
			Name:        naming.PortName(fmt.Sprintf("%s-grpc", o.name), defaultOTLPGRPCPort),
			Port:        o.config.Protocols.Grpc.getPortNumOrDefault(logger, defaultOTLPGRPCPort),
			TargetPort:  intstr.FromInt32(defaultOTLPGRPCPort),
			Protocol:    corev1.ProtocolTCP,
			AppProtocol: &grpc,
		})
	}
	if o.config.Protocols.HTTP != nil {
		ports = append(ports, corev1.ServicePort{
			Name:        naming.PortName(fmt.Sprintf("%s-http", o.name), defaultOTLPHTTPPort),
			Port:        o.config.Protocols.HTTP.getPortNumOrDefault(logger, defaultOTLPHTTPPort),
			TargetPort:  intstr.FromInt32(defaultOTLPHTTPPort),
			AppProtocol: &http,
		})
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
