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
package exporter

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// registry holds a record of all known exporter parsers.
var registry = make(map[string]parser.Builder)

// BuilderFor returns a parser builder for the given exporter name.
func BuilderFor(name string) parser.Builder {
	return registry[exporterType(name)]
}

// For returns a new parser for the given exporter name + config.
func For(logger logr.Logger, name string, config map[interface{}]interface{}) (parser.ComponentPortParser, error) {
	builder := BuilderFor(name)
	if builder == nil {
		return nil, fmt.Errorf("no builders for %s", name)
	}
	return builder(logger, name, config), nil
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

var (
	endpointKey = "endpoint"
)

func singlePortFromConfigEndpoint(logger logr.Logger, name string, config map[interface{}]interface{}) *corev1.ServicePort {
	endpoint := getAddressFromConfig(logger, name, endpointKey, config)

	switch e := endpoint.(type) {
	case nil:
		break
	case string:
		port, err := portFromEndpoint(e)
		if err != nil {
			logger.WithValues(endpointKey, e).Error(err, "couldn't parse the endpoint's port")
			return nil
		}

		return &corev1.ServicePort{
			Name: naming.PortName(name, port),
			Port: port,
		}
	default:
		logger.WithValues(endpointKey, endpoint).Error(fmt.Errorf("unrecognized type %T", endpoint), "exporter's endpoint isn't a string")
	}

	return nil
}

func getAddressFromConfig(logger logr.Logger, name, key string, config map[interface{}]interface{}) interface{} {
	endpoint, ok := config[key]
	if !ok {
		logger.V(2).Info("%s exporter doesn't have an %s", name, key)
		return nil
	}
	return endpoint
}

func portFromEndpoint(endpoint string) (int32, error) {
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
		return 0, errors.New("port should not be empty")
	}

	return int32(port), err
}

func exporterType(name string) string {
	// exporters have a name like:
	// - myexporter/custom
	// - myexporter
	// we extract the "myexporter" part and see if we have a parser for the exporter
	if strings.Contains(name, "/") {
		return name[:strings.Index(name, "/")]
	}

	return name
}
