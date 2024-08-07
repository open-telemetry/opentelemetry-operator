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

package components

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	_ Parser = &NopParser{}
)

// NopParser is a minimal processor mostly used for testing or coverage.
type NopParser struct {
	name string
}

func (n *NopParser) Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error) {
	return nil, nil
}

func (n *NopParser) GetRBACRules(logr.Logger, interface{}) ([]rbacv1.PolicyRule, error) {
	return nil, nil
}

func (n *NopParser) ParserType() string {
	return ComponentType(n.name)
}

func (n *NopParser) ParserName() string {
	return fmt.Sprintf("__%s", n.name)
}

func NewNopParser(name string, port int32, opts ...PortBuilderOption) *NopParser {
	return &NopParser{name: name}
}
