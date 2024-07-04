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

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

var _ parser.AuthzParser = &ResourceDetectionParser{}

const (
	parserNameResourceDetection = "__resourcedetection"
)

// ResourceDetectionParser parses the configuration for resourcedetection processor.
type ResourceDetectionParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewResourceDetectionParser builds a new parser for resourcedetection processor.
func NewResourceDetectionParser(logger logr.Logger, name string, config map[interface{}]interface{}) parser.AuthzParser {
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

func (o *ResourceDetectionParser) GetRBACRules() []authz.DynamicRolePolicy {
	var prs []rbacv1.PolicyRule

	detectorsCfg, ok := o.config["detectors"]
	if !ok {
		return []authz.DynamicRolePolicy{{Rules: prs}}
	}

	detectors, ok := detectorsCfg.([]interface{})
	if !ok {
		return []authz.DynamicRolePolicy{{Rules: prs}}
	}
	for _, d := range detectors {
		detectorName := fmt.Sprint(d)
		switch detectorName {
		case "k8snode":
			policy := rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list"},
			}
			prs = append(prs, policy)
		case "openshift":
			policy := rbacv1.PolicyRule{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"infrastructures", "infrastructures/status"},
				Verbs:     []string{"get", "watch", "list"},
			}
			prs = append(prs, policy)
		}
	}
	return []authz.DynamicRolePolicy{{Rules: prs}}
}

func init() {
	parser.AuthzRegister(parser.ComponentTypeProcessor, "resourcedetection", NewResourceDetectionParser)
}
