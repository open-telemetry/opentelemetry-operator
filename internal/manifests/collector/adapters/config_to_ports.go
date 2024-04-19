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

package adapters

import (
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

func GetComponentPorts(logger logr.Logger, parsers ...parser.ComponentPortParser) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	for _, cmptParser := range parsers {
		exprtPorts, err := cmptParser.Ports(logger)
		if err != nil {
			logger.Error(err, "parser for '%s' has returned an error: %w", cmptParser.ParserName(), err)
			continue
		}

		if len(exprtPorts) > 0 {
			ports = append(ports, exprtPorts...)
		}
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})

	return ports, nil
}
