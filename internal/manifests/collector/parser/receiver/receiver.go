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

package receiver

import (
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

// registry holds a record of all known receiver parsers.
var registry = make(map[string]parser.Builder)

// BuilderFor returns a parser builder for the given receiver name.
func BuilderFor(name string) parser.Builder {
	builder := registry[parser.ComponentType(name)]
	if builder == nil {
		builder = parser.NewSinglePortParser
	}

	return builder
}

// For returns a new parser for the given receiver name + config.
func For(name string, config interface{}) (parser.ComponentPortParser, error) {
	return BuilderFor(name)(name, config)
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
