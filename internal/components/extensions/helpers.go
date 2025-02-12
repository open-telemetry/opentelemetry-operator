// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// registry holds a record of all known receiver parsers.
var registry = make(map[string]components.Parser)

// Register adds a new parser builder to the list of known builders.
func Register(name string, p components.Parser) {
	registry[name] = p
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

// ParserFor returns a parser builder for the given exporter name.
func ParserFor(name string) components.Parser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	// We want the default for exporters to fail silently.
	return components.NewBuilder[any]().WithName(name).MustBuild()
}

var (
	componentParsers = []components.Parser{
		components.NewBuilder[healthcheckV1Config]().
			WithName("health_check").
			WithPort(13133).
			WithReadinessGen(HealthCheckV1Probe).
			WithLivenessGen(HealthCheckV1Probe).
			WithPortParser(func(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config healthcheckV1Config) ([]corev1.ServicePort, error) {
				return components.ParseSingleEndpointSilent(logger, name, defaultPort, &config.SingleEndpointConfig)
			}).
			MustBuild(),
		components.NewSinglePortParserBuilder("jaeger_query", 16686).
			WithTargetPort(16686).
			MustBuild(),
	}
)

func init() {
	for _, parser := range componentParsers {
		Register(parser.ParserType(), parser)
	}
}
