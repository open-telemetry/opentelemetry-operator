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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

var _ parsers.ComponentPortParser = &SyslogReceiverParser{}

const parserNameSyslog = "__syslog"

type SyslogReceiverConfig struct {
	Tcp *GenericReceiverConfig `json:"tcp,omitempty"`
	Udp *GenericReceiverConfig `json:"udp,omitempty"`
}

// SyslogReceiverParser parses the configuration for Syslog receivers.
type SyslogReceiverParser struct {
	config *SyslogReceiverConfig
	name   string
}

// NewSyslogReceiverParser builds a new parser for TCP log receivers.
func NewSyslogReceiverParser(name string, config interface{}) (parsers.ComponentPortParser, error) {
	c := &SyslogReceiverConfig{}
	if err := parsers.LoadMap[SyslogReceiverConfig](config, c); err != nil {
		return nil, err
	}
	if c.Tcp == nil && c.Udp == nil {
		return nil, errors.New("must set either udp or tcp")
	}
	return &SyslogReceiverParser{
		name:   name,
		config: c,
	}, nil
}

func (o *SyslogReceiverParser) Ports(logger logr.Logger) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	if o.config.Tcp != nil {
		port, err := o.config.Tcp.getPortNum()
		if err != nil {
			return nil, err
		}
		ports = append(ports, corev1.ServicePort{
			Name:     naming.PortName(o.name, port),
			Port:     port,
			Protocol: corev1.ProtocolTCP,
		})
	}
	if o.config.Udp != nil {
		port, err := o.config.Udp.getPortNum()
		if err != nil {
			return nil, err
		}
		ports = append(ports, corev1.ServicePort{
			Name:     naming.PortName(o.name, port),
			Port:     port,
			Protocol: corev1.ProtocolUDP,
		})
	}

	return ports, nil
}

func (o *SyslogReceiverParser) ParserName() string {
	return parserNameSyslog
}

func init() {
	Register("syslog", NewSyslogReceiverParser)
}
