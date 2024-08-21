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

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

type ParserOption[T any] func(*Option[T])

type Option[T any] struct {
	protocol    corev1.Protocol
	appProtocol *string
	targetPort  intstr.IntOrString
	nodePort    int32
	name        string
	port        int32
	portParser  PortParser[T]
	rbacGen     RBACRuleGenerator[T]
}

func NewEmptyOption[T any]() *Option[T] {
	return &Option[T]{}
}

func NewOption[T any](name string, port int32) *Option[T] {
	return &Option[T]{
		name: name,
		port: port,
	}
}

func (o *Option[T]) Apply(opts ...ParserOption[T]) {
	for _, opt := range opts {
		opt(o)
	}
}

func (o *Option[T]) GetServicePort() *corev1.ServicePort {
	return &corev1.ServicePort{
		Name:        naming.PortName(o.name, o.port),
		Port:        o.port,
		Protocol:    o.protocol,
		AppProtocol: o.appProtocol,
		TargetPort:  o.targetPort,
		NodePort:    o.nodePort,
	}
}

type Builder[T any] []ParserOption[T]

func NewBuilder[T any]() Builder[T] {
	return []ParserOption[T]{}
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
