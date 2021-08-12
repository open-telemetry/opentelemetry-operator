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

package adapters

import (
	"errors"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/parser"
)

var (
	// ErrNoReceivers indicates that there are no receivers in the configuration.
	ErrNoReceivers = errors.New("no receivers available as part of the configuration")

	// ErrReceiversNotAMap indicates that the receivers property isn't a map of values.
	ErrReceiversNotAMap = errors.New("receivers property in the configuration doesn't contain valid receivers")
)

// ConfigToReceiverPorts converts the incoming configuration object into a set of service ports required by the receivers.
func ConfigToReceiverPorts(logger logr.Logger, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	// now, we gather which ports we might need to open
	// for that, we get all the receivers and check their `endpoint` properties,
	// extracting the port from it. The port name has to be a "DNS_LABEL", so, we try to make it follow the pattern:
	// ${instance.Name}-${receiver.name}-${receiver.qualifier}
	// the receiver-name is typically the node name from the receivers map
	// the receiver-qualifier is what comes after the slash in the receiver name, but typically nil
	// examples:
	// ```yaml
	// receivers:
	//   examplereceiver:
	//     endpoint: 0.0.0.0:12345
	//   examplereceiver/settings:
	//     endpoint: 0.0.0.0:12346
	// in this case, we have two ports, named: "examplereceiver" and "examplereceiver-settings"
	receiversProperty, ok := config["receivers"]
	if !ok {
		return nil, ErrNoReceivers
	}

	receivers, ok := receiversProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrReceiversNotAMap
	}

	ports := []corev1.ServicePort{}
	for key, val := range receivers {
		receiver, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.Info("receiver doesn't seem to be a map of properties", "receiver", key)
			receiver = map[interface{}]interface{}{}
		}

		rcvrName := key.(string)
		rcvrParser := parser.For(logger, rcvrName, receiver)

		rcvrPorts, err := rcvrParser.Ports()
		if err != nil {
			// should we break the process and return an error, or just ignore this faulty parser
			// and let the other parsers add their ports to the service? right now, the best
			// option seems to be to log the failures and move on, instead of failing them all
			logger.Error(err, "parser for '%s' has returned an error: %w", rcvrName, err)
			continue
		}

		if len(rcvrPorts) > 0 {
			ports = append(ports, rcvrPorts...)
		}
	}
	return ports, nil
}
