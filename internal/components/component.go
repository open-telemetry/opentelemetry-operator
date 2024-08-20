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
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var (
	GrpcProtocol          = "grpc"
	HttpProtocol          = "http"
	UnsetPort       int32 = 0
	PortNotFoundErr       = errors.New("port should not be empty")
)

type PortRetriever interface {
	GetPortNum() (int32, error)
	GetPortNumOrDefault(logr.Logger, int32) int32
}

// PortParser is a function that returns a list of servicePorts given a config of type T.
type PortParser[T any] func(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config T) ([]corev1.ServicePort, error)

// RBACRuleGenerator is a function that generates a list of RBAC Rules given a configuration of type T
// It's expected that type T is the configuration used by a parser.
type RBACRuleGenerator[T any] func(logger logr.Logger, config T) ([]rbacv1.PolicyRule, error)
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

func WithRBACRuleGenerator[T any](r RBACRuleGenerator[T]) ParserOption[T] {
	return func(opt *Option[T]) {
		opt.rbacGen = r
	}
}

func WithPortParser[T any](p PortParser[T]) ParserOption[T] {
	return func(opt *Option[T]) {
		opt.portParser = p
	}
}

func WithTargetPort[T any](targetPort int32) ParserOption[T] {
	return func(opt *Option[T]) {
		opt.targetPort = intstr.FromInt32(targetPort)
	}
}

func WithAppProtocol[T any](proto *string) ParserOption[T] {
	return func(opt *Option[T]) {
		opt.appProtocol = proto
	}
}

func WithProtocol[T any](proto corev1.Protocol) ParserOption[T] {
	return func(opt *Option[T]) {
		opt.protocol = proto
	}
}

// ComponentType returns the type for a given component name.
// components have a name like:
// - mycomponent/custom
// - mycomponent
// we extract the "mycomponent" part and see if we have a parser for the component.
func ComponentType(name string) string {
	if strings.Contains(name, "/") {
		return name[:strings.Index(name, "/")]
	}
	return name
}

func PortFromEndpoint(endpoint string) (int32, error) {
	var err error
	var port int64

	r := regexp.MustCompile(":[0-9]+")

	if r.MatchString(endpoint) {
		portStr := r.FindString(endpoint)
		cleanedPortStr := strings.Replace(portStr, ":", "", -1)
		port, err = strconv.ParseInt(cleanedPortStr, 10, 32)

		if err != nil {
			return UnsetPort, err
		}
	}

	if port == 0 {
		return UnsetPort, PortNotFoundErr
	}

	return int32(port), err
}

type ParserRetriever func(string) Parser

type Parser interface {
	// Ports returns the service ports parsed based on the component's configuration where name is the component's name
	// of the form "name" or "type/name"
	Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error)

	// GetRBACRules returns the rbac rules for this component
	GetRBACRules(logger logr.Logger, config interface{}) ([]rbacv1.PolicyRule, error)

	// ParserType returns the type of this parser
	ParserType() string

	// ParserName is an internal name for the parser
	ParserName() string
}

func ConstructServicePort(current *corev1.ServicePort, port int32) corev1.ServicePort {
	svc := corev1.ServicePort{
		Name:        current.Name,
		Port:        port,
		NodePort:    current.NodePort,
		AppProtocol: current.AppProtocol,
		Protocol:    current.Protocol,
	}

	if port > 0 && current.TargetPort.IntValue() > 0 {
		svc.TargetPort = intstr.FromInt32(port)
	}
	return svc
}

func GetPortsForConfig(logger logr.Logger, config map[string]interface{}, retriever ParserRetriever) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	for componentName, componentDef := range config {
		parser := retriever(componentName)
		if parsedPorts, err := parser.Ports(logger, componentName, componentDef); err != nil {
			return nil, err
		} else {
			ports = append(ports, parsedPorts...)
		}
	}
	return ports, nil
}
