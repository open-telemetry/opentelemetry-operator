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

package parser

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
)

// AuthzParser specifies the methods to implement to parse a processor.
type AuthzParser interface {
	ParserName() string
	GetRBACRules() []authz.DynamicRolePolicy
}

// AuthzBuilder specifies the signature required for parser builders.
type AuthzBuilder func(logr.Logger, string, map[interface{}]interface{}) AuthzParser

// authzTypedRegistry holds a record of all known component parsers.
var authzTypedRegistry = make(map[ComponentType]map[string]AuthzBuilder)

// AuthzBuilderFor returns a parser builder for the given processor name.
func AuthzBuilderFor(typ ComponentType, name string) AuthzBuilder {
	if builders, ok := authzTypedRegistry[typ]; ok {
		return builders[ComponentName(name)]
	}
	return nil
}

// AuthzFor returns a new parser for the given component type + name + config.
func AuthzFor(logger logr.Logger, typ ComponentType, name string, config map[interface{}]interface{}) (AuthzParser, error) {
	builder := AuthzBuilderFor(typ, name)
	if builder == nil {
		return nil, fmt.Errorf("no builders for %s", name)
	}
	return builder(logger, name, config), nil
}

// AuthzRegister adds a new parser builder to the list of known builders.
func AuthzRegister(typ ComponentType, name string, builder AuthzBuilder) {
	if authzTypedRegistry[typ] == nil {
		authzTypedRegistry[typ] = make(map[string]AuthzBuilder)
	}
	authzTypedRegistry[typ][name] = builder
}
