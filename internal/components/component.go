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
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	unsetPort = 0
)

var (
	portNotFoundErr = errors.New("port should not be empty")
	grpc            = "grpc"
	http            = "http"
)

type PortRetriever interface {
	GetPortNum() (int32, error)
	GetPortNumOrDefault(logr.Logger, int32) int32
}

type PortBuilderOption func(portBuilder *corev1.ServicePort)

func WithTargetPort(targetPort int32) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.TargetPort = intstr.FromInt32(targetPort)
	}
}
func WithNodePort(nodePort int32) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.NodePort = nodePort
	}
}

func WithAppProtocol(proto *string) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.AppProtocol = proto
	}
}

func WithProtocol(proto corev1.Protocol) PortBuilderOption {
	return func(servicePort *corev1.ServicePort) {
		servicePort.Protocol = proto
	}
}

func ComponentType(name string) string {
	// components have a name like:
	// - mycomponent/custom
	// - mycomponent
	// we extract the "mycomponent" part and see if we have a parser for the component
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
		port, err = strconv.ParseInt(strings.Replace(r.FindString(endpoint), ":", "", -1), 10, 32)

		if err != nil {
			return 0, err
		}
	}

	if port == 0 {
		return 0, portNotFoundErr
	}

	return int32(port), err
}

type ComponentPortParser interface {
	// Ports returns the service ports parsed based on the exporter's configuration
	Ports(logger logr.Logger, config interface{}) ([]corev1.ServicePort, error)

	// ParserType returns the name of this parser
	ParserType() string

	// ParserName is an internal name for the parser
	ParserName() string
}

// registry holds a record of all known receiver parsers.
var registry = make(map[string]ComponentPortParser)

// Register adds a new parser builder to the list of known builders.
func Register(name string, p ComponentPortParser) {
	registry[name] = p
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

// BuilderFor returns a parser builder for the given exporter name.
func BuilderFor(name string) ComponentPortParser {
	if parser, ok := registry[ComponentType(name)]; ok {
		return parser
	}
	return NewSinglePortParser(ComponentType(name), unsetPort)
}

func LoadMap[T any](m interface{}, in T) error {
	// Convert map to JSON bytes
	yamlData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	// Unmarshal YAML into the provided struct
	if err := json.Unmarshal(yamlData, in); err != nil {
		return err
	}
	return nil
}

func ConstructServicePort(current *corev1.ServicePort, port int32) corev1.ServicePort {
	return corev1.ServicePort{
		Name:        current.Name,
		Port:        port,
		TargetPort:  current.TargetPort,
		NodePort:    current.NodePort,
		AppProtocol: current.AppProtocol,
		Protocol:    current.Protocol,
	}
}

func init() {
	parsers := append(scraperReceivers, append(singleEndpointConfigs, multiPortReceivers...)...)
	for _, parser := range parsers {
		Register(parser.ParserType(), parser)
	}
}
