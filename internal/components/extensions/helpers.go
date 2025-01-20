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

package extensions

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// registry holds a record of all known receiver parsers.
var registry = map[string]components.Parser{
	"health_check": components.NewBuilder[healthcheckV1Config]().
		WithName("health_check").
		WithPort(13133).
		WithReadinessGen(healthCheckV1Probe).
		WithLivenessGen(healthCheckV1Probe).
		WithPortParser(func(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config healthcheckV1Config) ([]corev1.ServicePort, error) {
			return components.ParseSingleEndpointSilent(logger, name, defaultPort, &config.SingleEndpointConfig)
		}).
		MustBuild(),
	"jaeger_query": components.NewSinglePortParserBuilder("jaeger_query", 16686).
		WithTargetPort(16686).
		MustBuild(),
}

// ParserFor returns a parser builder for the given exporter name.
func ParserFor(name string) components.Parser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	// We want the default for exporters to fail silently.
	return components.NewBuilder[any]().WithName(name).MustBuild()
}
