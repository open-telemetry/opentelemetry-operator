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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Builder[T any] []ParserOption[T]

func NewBuilder[T any]() Builder[T] {
	return make([]ParserOption[T], 8)
}

func (b Builder[T]) WithProtocol(protocol corev1.Protocol) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.protocol = protocol
	})
}
func (b Builder[T]) WithAppProtocol(appProtocol *string) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.appProtocol = appProtocol
	})
}
func (b Builder[T]) WithTargetPort(targetPort int32) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.targetPort = intstr.FromInt32(targetPort)
	})
}
func (b Builder[T]) WithNodePort(nodePort int32) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.nodePort = nodePort
	})
}
func (b Builder[T]) WithName(name string) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.name = name
	})
}
func (b Builder[T]) WithPort(port int32) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.port = port
	})
}
func (b Builder[T]) WithPortParser(portParser PortParser[T]) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.portParser = portParser
	})
}
func (b Builder[T]) WithRbacGen(rbacGen RBACRuleGenerator[T]) Builder[T] {
	return append(b, func(o *Option[T]) {
		o.rbacGen = rbacGen
	})
}

func (b Builder[T]) Build() (*GenericParser[T], error) {
	o := NewEmptyOption[T]()
	o.Apply(b...)
	if len(o.name) == 0 {
		return nil, fmt.Errorf("invalid option struct, no name specified")
	}
	return &GenericParser[T]{name: o.name, portParser: o.portParser, rbacGen: o.rbacGen, option: o}, nil
}

func (b Builder[T]) MustBuild() *GenericParser[T] {
	if p, err := b.Build(); err != nil {
		panic(err)
	} else {
		return p
	}
}
