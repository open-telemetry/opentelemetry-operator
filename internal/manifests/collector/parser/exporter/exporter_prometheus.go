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

package exporter

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var _ parser.ComponentPortParser = &PrometheusExporterParser{}

const (
	parserNamePrometheus  = "__prometheus"
	defaultPrometheusPort = 8888
)

// PrometheusExporterParser parses the configuration for OTLP receivers.
type PrometheusExporterParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewPrometheusExporterParser builds a new parser for OTLP receivers.
func NewPrometheusExporterParser(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ComponentPortParser {
	return &PrometheusExporterParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

// Ports returns all the service ports for all protocols in this parser.
func (o *PrometheusExporterParser) Ports() ([]corev1.ServicePort, error) {
	ports := []corev1.ServicePort{}
	if o.config == nil {
		ports = append(ports,
			corev1.ServicePort{
				Name:       naming.PortName(o.name, defaultPrometheusPort),
				Port:       defaultPrometheusPort,
				TargetPort: intstr.FromInt(int(defaultPrometheusPort)),
				Protocol:   corev1.ProtocolTCP,
			},
		)
	} else {
		ports = append(
			ports, *singlePortFromConfigEndpoint(o.logger, o.name, o.config),
		)
	}

	return ports, nil
}

// ParserName returns the name of this parser.
func (o *PrometheusExporterParser) ParserName() string {
	return parserNamePrometheus
}

func init() {
	Register("prometheus", NewPrometheusExporterParser)
}
