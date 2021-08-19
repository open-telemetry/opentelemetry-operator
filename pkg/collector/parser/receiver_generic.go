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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

const parserNameGeneric = "__generic"

var _ ReceiverParser = &GenericReceiver{}

// GenericReceiver is a special parser for generic receivers. It doesn't self-register and should be created/used directly.
type GenericReceiver struct {
	logger      logr.Logger
	name        string
	config      map[interface{}]interface{}
	defaultPort int32
	parserName  string
}

// NOTE: Operator will sync with only receivers that aren't scrapers. Operator sync up receivers
// so that it can expose the required port based on the receiver's config. Receiver scrapers are ignored.

// NewGenericReceiverParser builds a new parser for generic receivers.
func NewGenericReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:     logger,
		name:       name,
		config:     config,
		parserName: parserNameGeneric,
	}
}

// Ports returns all the service ports for all protocols in this parser.
func (g *GenericReceiver) Ports() ([]corev1.ServicePort, error) {
	port := singlePortFromConfigEndpoint(g.logger, g.name, g.config)
	if port != nil {
		return []corev1.ServicePort{*port}, nil
	}

	if g.defaultPort > 0 {
		return []corev1.ServicePort{{
			Port: g.defaultPort,
			Name: portName(g.name, g.defaultPort),
		}}, nil
	}

	return []corev1.ServicePort{}, nil
}

// ParserName returns the name of this parser.
func (g *GenericReceiver) ParserName() string {
	return g.parserName
}
