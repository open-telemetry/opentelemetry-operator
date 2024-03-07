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

var _ parser.ComponentPortParser = &TcpLogReceiverParser{}

const parserNameTcpLog = "__tcplog"

// TcpLogReceiverParser parses the configuration for TCP log receivers.
type TcpLogReceiverParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewTcpLogReceiverParser builds a new parser for TCP log receivers.
func NewTcpLogReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ComponentPortParser {
	return &TcpLogReceiverParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

func (o *TcpLogReceiverParser) Ports() ([]corev1.ServicePort, error) {
	// tcplog receiver hold the endpoint value in `listen_address` field
	var endpoint = getAddressFromConfig(o.logger, o.name, listenAddressKey, o.config)

	switch e := endpoint.(type) {
	case nil:
		break
	case string:
		port, err := portFromEndpoint(e)
		if err != nil {
			o.logger.WithValues(listenAddressKey, e).Error(err, "couldn't parse the endpoint's port")
			return nil, nil
		}

		return []corev1.ServicePort{{
			Port:     port,
			Name:     naming.PortName(o.name, port),
			Protocol: corev1.ProtocolTCP,
		}}, nil
	default:
		o.logger.WithValues(listenAddressKey, endpoint).Error(fmt.Errorf("unrecognized type %T", endpoint), "receiver's endpoint isn't a string")
	}

	return []corev1.ServicePort{}, nil
}

func (o *TcpLogReceiverParser) ParserName() string {
	return parserNameTcpLog
}

func init() {
	Register("tcplog", NewTcpLogReceiverParser)
}
