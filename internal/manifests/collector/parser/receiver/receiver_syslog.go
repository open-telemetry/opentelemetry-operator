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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var _ parser.ComponentPortParser = &SyslogReceiverParser{}

const parserNameSyslog = "__syslog"

// SyslogReceiverParser parses the configuration for TCP log receivers.
type SyslogReceiverParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewSyslogReceiverParser builds a new parser for TCP log receivers.
func NewSyslogReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ComponentPortParser {
	return &SyslogReceiverParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

func (o *SyslogReceiverParser) Ports() ([]corev1.ServicePort, error) {
	var endpoint interface{}
	var endpointName string
	var protocol corev1.Protocol
	var c map[interface{}]interface{}

	// syslog receiver contains the endpoint
	// that needs to be exposed one level down inside config
	// i.e. either in tcp or udp section with field key
	// as `listen_address`
	if tcp, isTCP := o.config["tcp"]; isTCP && tcp != nil {
		c = tcp.(map[interface{}]interface{})
		endpointName = "tcp"
		endpoint = getAddressFromConfig(o.logger, o.name, listenAddressKey, c)
		protocol = corev1.ProtocolTCP
	} else if udp, isUDP := o.config["udp"]; isUDP && udp != nil {
		c = udp.(map[interface{}]interface{})
		endpointName = "udp"
		endpoint = getAddressFromConfig(o.logger, o.name, listenAddressKey, c)
		protocol = corev1.ProtocolUDP
	}

	switch e := endpoint.(type) {
	case nil:
		break
	case string:
		port, err := portFromEndpoint(e)
		if err != nil {
			o.logger.WithValues(listenAddressKey, e).Error(err, fmt.Sprintf("couldn't parse the %s endpoint's port", endpointName))
			return nil, nil
		}

		return []corev1.ServicePort{{
			Port:     port,
			Name:     naming.PortName(o.name, port),
			Protocol: protocol,
		}}, nil
	default:
		o.logger.WithValues(listenAddressKey, endpoint).Error(fmt.Errorf("unrecognized type %T of %s endpoint", endpoint, endpointName),
			"receiver's endpoint isn't a string")
	}

	return []corev1.ServicePort{}, nil
}

func (o *SyslogReceiverParser) ParserName() string {
	return parserNameSyslog
}

func init() {
	Register("syslog", NewSyslogReceiverParser)
}
