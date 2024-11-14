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
	components.NewBuilder[K8sAttributeConfig]().
		WithName("k8sattributes").
		WithClusterRoleRulesGen(generateK8SAttrClusterRoleRules).
		MustBuild(),
	components.NewBuilder[ResourceDetectionConfig]().
		WithName("resourcedetection").
		WithClusterRoleRulesGen(generateResourceDetectionClusterRoleRules).
		MustBuild(),
}

func init() {
	for _, parser := range componentParsers {
		Register(parser.ParserType(), parser)
	}
}
