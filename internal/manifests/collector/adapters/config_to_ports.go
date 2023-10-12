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
	"net"
	"sort"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"

	exporterParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/exporter"
	receiverParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/receiver"
)

var (
	// ErrNoExporters indicates that there are no exporters in the configuration.
	ErrNoExporters = errors.New("no exporters available as part of the configuration")

	// ErrNoReceivers indicates that there are no receivers in the configuration.
	ErrNoReceivers = errors.New("no receivers available as part of the configuration")

	// ErrReceiversNotAMap indicates that the receivers property isn't a map of values.
	ErrReceiversNotAMap = errors.New("receivers property in the configuration doesn't contain valid receivers")

	// ErrExportersNotAMap indicates that the exporters property isn't a map of values.
	ErrExportersNotAMap = errors.New("exporters property in the configuration doesn't contain valid exporters")
)

// ConfigToExporterPorts converts the incoming configuration object into a set of service ports required by the exporters.
func ConfigToExporterPorts(logger logr.Logger, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	// now, we gather which ports we might need to open
	// for that, we get all the exporters and check their `endpoint` properties,
	// extracting the port from it. The port name has to be a "DNS_LABEL", so, we try to make it follow the pattern:
	// ${instance.Name}-${exporter.name}-${exporter.qualifier}
	// the exporter-name is typically the node name from the exporters map
	// the exporter-qualifier is what comes after the slash in the exporter name, but typically nil
	// examples:
	// ```yaml
	// exporters:
	//   exampleexporter:
	//     endpoint: 0.0.0.0:12345
	//   exampleexporter/settings:
	//     endpoint: 0.0.0.0:12346
	// in this case, we have two ports, named: "exampleexporter" and "exampleexporter-settings"
	exportersProperty, ok := config["exporters"]
	if !ok {
		return nil, ErrNoExporters
	}
	expEnabled := GetEnabledExporters(logger, config)
	if expEnabled == nil {
		return nil, ErrExportersNotAMap
	}
	exporters, ok := exportersProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrExportersNotAMap
	}

	ports := []corev1.ServicePort{}
	for key, val := range exporters {
		// This check will pass only the enabled exporters,
		// then only the related ports will be opened.
		if !expEnabled[key] {
			continue
		}
		exporter, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.V(2).Info("exporter doesn't seem to be a map of properties", "exporter", key)
			exporter = map[interface{}]interface{}{}
		}

		exprtName := key.(string)
		exprtParser, err := exporterParser.For(logger, exprtName, exporter)
		if err != nil {
			logger.V(2).Info("no parser found for '%s'", exprtName)
			continue
		}

		exprtPorts, err := exprtParser.Ports()
		if err != nil {
			logger.Error(err, "parser for '%s' has returned an error: %w", exprtName, err)
			continue
		}

		if len(exprtPorts) > 0 {
			ports = append(ports, exprtPorts...)
		}
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports, nil
}

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
	recEnabled := GetEnabledReceivers(logger, config)
	if recEnabled == nil {
		return nil, ErrReceiversNotAMap
	}
	receivers, ok := receiversProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrReceiversNotAMap
	}

	ports := []corev1.ServicePort{}
	for key, val := range receivers {
		// This check will pass only the enabled receivers,
		// then only the related ports will be opened.
		if !recEnabled[key] {
			continue
		}
		receiver, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.Info("receiver doesn't seem to be a map of properties", "receiver", key)
			receiver = map[interface{}]interface{}{}
		}

		rcvrName := key.(string)
		rcvrParser := receiverParser.For(logger, rcvrName, receiver)

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

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports, nil
}

func ConfigToPorts(logger logr.Logger, config map[interface{}]interface{}) []corev1.ServicePort {
	ports, err := ConfigToReceiverPorts(logger, config)
	if err != nil {
		logger.Error(err, "there was a problem while getting the ports from the receivers")
	}

	exporterPorts, err := ConfigToExporterPorts(logger, config)
	if err != nil {
		logger.Error(err, "there was a problem while getting the ports from the exporters")
	}
	ports = append(ports, exporterPorts...)

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports
}

// ConfigToMetricsPort gets the port number for the metrics endpoint from the collector config if it has been set.
func ConfigToMetricsPort(logger logr.Logger, config map[interface{}]interface{}) (int32, error) {
	// we don't need to unmarshal the whole config, just follow the keys down to
	// the metrics address.
	type metricsCfg struct {
		Address string
	}
	type telemetryCfg struct {
		Metrics metricsCfg
	}
	type serviceCfg struct {
		Telemetry telemetryCfg
	}
	type cfg struct {
		Service serviceCfg
	}
	var cOut cfg
	err := mapstructure.Decode(config, &cOut)
	if err != nil {
		return 0, err
	}

	_, port, err := net.SplitHostPort(cOut.Service.Telemetry.Metrics.Address)
	if err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(i64), nil
}
