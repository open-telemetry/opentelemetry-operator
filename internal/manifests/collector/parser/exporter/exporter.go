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

// Package parser is for parsing the OpenTelemetry Collector configuration.
package exporter

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

// registry holds a record of all known exporter parsers.
var registry = make(map[string]parser.Builder)

// BuilderFor returns a parser builder for the given exporter name.
func BuilderFor(name string) parser.Builder {
	return registry[parser.ComponentType(name)]
}

// For returns a new parser for the given exporter name + config.
func For(name string, config interface{}) (parser.ComponentPortParser, error) {
	builder := BuilderFor(name)
	if builder == nil {
		return nil, fmt.Errorf("no builders for %s", name)
	}
	return builder(name, config)
}

// Register adds a new parser builder to the list of known builders.
func Register(name string, builder parser.Builder) {
	registry[name] = builder
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}
