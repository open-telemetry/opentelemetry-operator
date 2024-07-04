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
package processor

// ProcessorParser specifies the methods to implement to parse a processor.
//type ProcessorParser interface {
//	ParserName() string
//	GetRBACRules() []authz.DynamicRolePolicy
//}

// Builder specifies the signature required for parser builders.
//type Builder func(logr.Logger, string, map[interface{}]interface{}) ProcessorParser

// registry holds a record of all known processor parsers.
//var registry = make(map[string]Builder)

// BuilderFor returns a parser builder for the given processor name.
//func BuilderFor(name string) Builder {
//	return registry[parser.ComponentType(name)]
//}

// For returns a new parser for the given processor name + config.
//func For(logger logr.Logger, name string, config map[interface{}]interface{}) (ProcessorParser, error) {
//	builder := BuilderFor(name)
//	if builder == nil {
//		return nil, fmt.Errorf("no builders for %s", name)
//	}
//	return builder(logger, name, config), nil
//}

//// Register adds a new parser builder to the list of known builders.
//func Register(name string, builder Builder) {
//	registry[name] = builder
//}
//
//// IsRegistered checks whether a parser is registered with the given name.
//func IsRegistered(name string) bool {
//	_, ok := registry[name]
//	return ok
//}
