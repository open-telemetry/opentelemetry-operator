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

type ComponentPortParser interface {
	// Ports returns the service ports parsed based on the exporter's configuration
	Ports() ([]corev1.ServicePort, error)

	// ParserName returns the name of this parser
	ParserName() string
}

// Builder specifies the signature required for parser builders.
type Builder func(logr.Logger, string, map[interface{}]interface{}) ComponentPortParser
