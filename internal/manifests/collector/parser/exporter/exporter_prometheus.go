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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

const (
	defaultPrometheusPort = 8888
)

// NewPrometheusExporterParser builds a new parser for OTLP receivers.
func NewPrometheusExporterParser(name string, config interface{}) (parser.ComponentPortParser, error) {
	return parser.CreateParser(
		parser.WithSinglePort(defaultPrometheusPort,
			parser.WithTargetPort(defaultPrometheusPort),
			parser.WithProtocol(corev1.ProtocolTCP)),
	)(name, config)
}

func init() {
	Register("prometheus", NewPrometheusExporterParser)
}
