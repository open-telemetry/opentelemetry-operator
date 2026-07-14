// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processors

import "github.com/open-telemetry/opentelemetry-operator/internal/components"

// registry holds a record of all known receiver parsers.
var registry = make(map[string]components.Parser)

// Register adds a new parser builder to the list of known builders.
func Register(name string, p components.Parser) {
	registry[name] = p
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[components.ComponentType(name)]
	return ok
}

// ProcessorFor returns a parser builder for the given exporter name.
func ProcessorFor(name string) components.Parser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	return components.NewBuilder[any]().WithName(name).MustBuild()
}

var componentParsers = []components.Parser{
	// k8s_attributes, formerly k8sattributes
	// (open-telemetry/opentelemetry-collector-contrib#45901, see #4922).
	components.NewBuilder[K8sAttributeConfig]().WithName("k8s_attributes").WithRbacGen(GenerateK8SAttrRbacRules).WithAlias("k8sattributes").MustBuild(),
	// resource_detection, formerly resourcedetection
	// (open-telemetry/opentelemetry-collector-contrib#48525).
	components.NewBuilder[ResourceDetectionConfig]().WithName("resource_detection").WithRbacGen(GenerateResourceDetectionRbacRules).WithAlias("resourcedetection").MustBuild(),
}

func init() {
	for _, parser := range componentParsers {
		Register(parser.ParserType(), parser)
		for _, alias := range parser.ParserAliases() {
			Register(alias, parser)
		}
	}
}
