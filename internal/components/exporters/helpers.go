// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporters

import (
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// Registry holds a record of all known exporter parsers.
var Registry = map[string]components.Parser{
	"prometheus": components.NewSinglePortParserBuilder("prometheus", 8888).MustBuild(),
}

// GetParser returns a parser builder for the given exporter name.
func GetParser(name string) components.Parser {
	return components.GetParser(name, Registry)
}
