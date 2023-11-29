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
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	exporterParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/exporter"
	receiverParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/receiver"
)

type ComponentType int

const (
	ComponentTypeReceiver ComponentType = iota
	ComponentTypeExporter
)

func (c ComponentType) String() string {
	return [...]string{"receiver", "exporter"}[c]
}

// ConfigToComponentPorts converts the incoming configuration object into a set of service ports required by the exporters.
func ConfigToComponentPorts(logger logr.Logger, cType ComponentType, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	// now, we gather which ports we might need to open
	// for that, we get all the exporters and check their `endpoint` properties,
	// extracting the port from it. The port name has to be a "DNS_LABEL", so, we try to make it follow the pattern:
	// ${instance.Name}-${exporter.name}-${exporter.qualifier}
	// the exporter-name is typically the node name from the exporters map
	// the exporter-qualifier is what comes after the slash in the exporter name, but typically nil
	// examples:
	// ```yaml
	// components:
	//   componentexample:
	//     endpoint: 0.0.0.0:12345
	//   componentexample/settings:
	//     endpoint: 0.0.0.0:12346
	// in this case, we have 2 ports, named: "componentexample" and "componentexample-settings"
	componentsProperty, ok := config[fmt.Sprintf("%ss", cType.String())]
	if !ok {
		return nil, fmt.Errorf("no %ss available as part of the configuration", cType)
	}

	components, ok := componentsProperty.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("%ss doesn't contain valid components", cType.String())
	}

	compEnabled := getEnabledComponents(config, cType)

	if compEnabled == nil {
		return nil, fmt.Errorf("no enabled %ss available as part of the configuration", cType)
	}

	ports := []corev1.ServicePort{}
	for key, val := range components {
		// This check will pass only the enabled components,
		// then only the related ports will be opened.
		if !compEnabled[key] {
			continue
		}
		exporter, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.V(2).Info("component doesn't seem to be a map of properties", cType.String(), key)
			exporter = map[interface{}]interface{}{}
		}

		cmptName := key.(string)
		var cmptParser parser.ComponentPortParser
		var err error
		switch cType {
		case ComponentTypeExporter:
			cmptParser, err = exporterParser.For(logger, cmptName, exporter)
		case ComponentTypeReceiver:
			cmptParser, err = receiverParser.For(logger, cmptName, exporter)
		}

		if err != nil {
			logger.V(2).Info("no parser found for '%s'", cmptName)
			continue
		}

		exprtPorts, err := cmptParser.Ports()
		if err != nil {
			logger.Error(err, "parser for '%s' has returned an error: %w", cmptName, err)
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

func ConfigToPorts(logger logr.Logger, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	ports, err := ConfigToComponentPorts(logger, ComponentTypeReceiver, config)
	if err != nil {
		logger.Error(err, "there was a problem while getting the ports from the receivers")
		return nil, err
	}

	exporterPorts, err := ConfigToComponentPorts(logger, ComponentTypeExporter, config)
	if err != nil {
		logger.Error(err, "there was a problem while getting the ports from the exporters")
		return nil, err
	}

	ports = append(ports, exporterPorts...)

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports, nil
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

	_, port, netErr := net.SplitHostPort(cOut.Service.Telemetry.Metrics.Address)
	if netErr != nil && strings.Contains(netErr.Error(), "missing port in address") {
		return 8888, nil
	} else if netErr != nil {
		return 0, netErr
	}
	i64, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(i64), nil
}
