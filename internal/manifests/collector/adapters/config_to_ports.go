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
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	exporterParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/exporter"
	receiverParser "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/receiver"
)

type ComponentType int

const (
	ComponentTypeReceiver ComponentType = iota
	ComponentTypeExporter
	ComponentTypeProcessor
)

func (c ComponentType) String() string {
	return [...]string{"receiver", "exporter", "processor"}[c]
}

func PortsForExporters(l logr.Logger, c v1beta1.Config) ([]corev1.ServicePort, error) {
	compEnabled := getEnabledComponents(c.Service, ComponentTypeExporter)
	return componentPorts(l, c.Exporters, exporterParser.For, compEnabled)
}

func PortsForReceivers(l logr.Logger, c v1beta1.Config) ([]corev1.ServicePort, error) {
	compEnabled := getEnabledComponents(c.Service, ComponentTypeReceiver)
	return componentPorts(l, c.Receivers, receiverParser.For, compEnabled)
}

func componentPorts(l logr.Logger, c v1beta1.AnyConfig, p parser.For, enabledComponents map[string]bool) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	for cmptName, val := range c.Object {
		if !enabledComponents[cmptName] {
			continue
		}
		component, ok := val.(map[string]interface{})
		if !ok {
			component = map[string]interface{}{}
		}
		componentParser, err := p(l, cmptName, component)
		if err != nil {
			l.Error(err, "failed to retrieve parser for '%s', has returned an error: %w", cmptName, err)
			continue
		}
		componentPorts, err := componentParser.Ports()
		if err != nil {
			l.Error(err, "parser for '%s' has returned an error: %w", cmptName, err)
			continue
		}
		ports = append(ports, componentPorts...)
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports, nil
}

func ConfigToPorts(logger logr.Logger, config v1beta1.Config) ([]corev1.ServicePort, error) {
	ports, err := PortsForReceivers(logger, config)
	if err != nil {
		logger.Error(err, "there was a problem while getting the ports from the receivers")
		return nil, err
	}

	exporterPorts, err := PortsForExporters(logger, config)
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
