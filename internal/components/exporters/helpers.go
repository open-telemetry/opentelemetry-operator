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
	// Default exporter parser applies TLS profile defaults but returns no ports.
	return NewExporterParser(name)
}
