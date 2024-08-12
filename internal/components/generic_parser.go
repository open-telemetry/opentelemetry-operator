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
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
)

var (
	_ Parser = &GenericParser[SingleEndpointConfig]{}
)

// GenericParser serves as scaffolding for custom parsing logic by isolating
// functionality to idempotent functions.
type GenericParser[T any] struct {
	name       string
	portParser PortParser[T]
	option     *Option
}

func (g *GenericParser[T]) Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error) {
	if g.portParser == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.portParser(logger, name, g.option, parsed)
}

func (g *GenericParser[T]) ParserType() string {
	return ComponentType(g.name)
}

func (g *GenericParser[T]) ParserName() string {
	return fmt.Sprintf("__%s", g.name)
}

func NewGenericParser[T any](name string, port int32, parser PortParser[T], opts ...PortBuilderOption) *GenericParser[T] {
	o := NewOption(name, port)
	o.Apply(opts...)
	return &GenericParser[T]{name: name, portParser: parser, option: o}
}
