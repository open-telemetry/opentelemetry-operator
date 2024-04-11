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
	"strings"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

var _ ProcessorParser = &K8sAttributesParser{}

const (
	parserNameK8sAttributes = "__k8sattributes"
)

// PrometheusExporterParser parses the configuration for k8sattributes processor.
type K8sAttributesParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewK8sAttributesParser builds a new parser k8sattributes processor.
func NewK8sAttributesParser(logger logr.Logger, name string, config map[interface{}]interface{}) ProcessorParser {
	return &K8sAttributesParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

// ParserName returns the name of this parser.
func (o *K8sAttributesParser) ParserName() string {
	return parserNameK8sAttributes
}

func (o *K8sAttributesParser) GetRBACRules() []rbacv1.PolicyRule {
	// These policies need to be added always
	var prs []rbacv1.PolicyRule = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "namespaces"},
			Verbs:     []string{"get", "watch", "list"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{"replicasets"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	extractCfg, ok := o.config["extract"]
	if !ok {
		return prs
	}

	metadataCfg, ok := extractCfg.(map[interface{}]interface{})["metadata"]
	if !ok {
		return prs
	}

	metadata, ok := metadataCfg.([]interface{})
	if !ok {
		return prs
	}

	for _, m := range metadata {
		metadataField := fmt.Sprint(m)
		if strings.Contains(metadataField, "k8s.node") {
			prs = append(prs,
				rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "watch", "list"},
				},
			)
		}
	}

	return prs
}

func init() {
	Register("k8sattributes", NewK8sAttributesParser)
}
