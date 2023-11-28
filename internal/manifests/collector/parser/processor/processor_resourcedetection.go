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

package processor

import (
	"fmt"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ ProcessorParser = &ResourceDetectionParser{}

const (
	parserNameResourceDetection  = "__resourcedetection"
)

// PrometheusExporterParser parses the configuration for OTLP receivers.
type ResourceDetectionParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewPrometheusExporterParser builds a new parser for OTLP receivers.
func NewResourceDetectionParser(logger logr.Logger, name string, config map[interface{}]interface{}) ProcessorParser {
	return &ResourceDetectionParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

// ParserName returns the name of this parser.
func (o *ResourceDetectionParser) ParserName() string {
	return parserNameResourceDetection
}

func (o* ResourceDetectionParser) GetRBACRules() []rbacv1.PolicyRule {
	var prs []rbacv1.PolicyRule

	detectorsCfg, ok := o.config["detectors"]
	if !ok {
		return prs
	}

	detectors, ok := detectorsCfg.([]interface{})
	if !ok {
		return prs
	}
	for _, d := range detectors {
		detectorName := fmt.Sprint(d)
		switch detectorName{
		case "openshift":
			policy := rbacv1.PolicyRule{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"infrastructures", "infrastructures/status"},
				Verbs: []string{"get", "watch", "list"},
			}
			prs = append(prs,policy)
		}
	}
	return prs
}

func init() {
	Register("resourcedetection", NewResourceDetectionParser)
}
