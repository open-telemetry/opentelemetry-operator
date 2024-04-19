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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

type SyslogReceiverConfig struct {
	Tcp *parser.SingleEndpointConfig `json:"tcp,omitempty"`
	Udp *parser.SingleEndpointConfig `json:"udp,omitempty"`
}

func syslogReceiverConfigFactory() *SyslogReceiverConfig {
	return &SyslogReceiverConfig{}
}

func (s SyslogReceiverConfig) configByProtocol() map[string]*parser.SingleEndpointConfig {
	if s.Tcp != nil {
		return map[string]*parser.SingleEndpointConfig{
			"tcp": s.Tcp,
		}
	}
	return map[string]*parser.SingleEndpointConfig{
		"udp": s.Udp,
	}
}

var baseSyslogConf = []MultiPortOption{
	WithPortMapping(
		"tcp",
		unsetPort,
		parser.WithProtocol(corev1.ProtocolTCP),
	), WithPortMapping(
		"udp",
		unsetPort,
		parser.WithProtocol(corev1.ProtocolUDP),
	),
}

// NewSyslogReceiverParser builds a new parser for TCP log receivers.
func NewSyslogReceiverParser(name string, config interface{}) (parser.ComponentPortParser, error) {
	return createMultiPortParser(syslogReceiverConfigFactory, baseSyslogConf...)(name, config)
}

func init() {
	Register("syslog", NewSyslogReceiverParser)
}
