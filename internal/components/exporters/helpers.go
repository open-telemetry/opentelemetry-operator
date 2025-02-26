// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporters

import (
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// registry holds a record of all known receiver parsers.
var registry = map[string]components.Parser{
	"prometheus": components.NewSinglePortParserBuilder("prometheus", 8888).MustBuild(),
}

// ParserFor returns a parser builder for the given exporter name.
func ParserFor(name string) components.Parser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	// We want the default for exporters to fail silently.
	return components.NewBuilder[any]().WithName(name).MustBuild()
}
