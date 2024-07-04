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

package exporter

import (
	"strings"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

var _ parser.AuthzParser = &LBExporterParser{}

const (
	parserNameLoadBalancing = "__loadbalancing"
)

// LBExporterParser parses the configuration for loadbalancing exporters.
type LBExporterParser struct {
	config map[any]any
	logger logr.Logger
	name   string
}

// NewLBExporterParser builds a new parser loadbalancing exporters.
func NewLBExporterParser(logger logr.Logger, name string, config map[any]any) parser.AuthzParser {
	return &LBExporterParser{
		logger: logger,
		name:   name,
		config: config,
	}
}

func (L LBExporterParser) ParserName() string {
	return parserNameLoadBalancing
}

func (L LBExporterParser) GetRBACRules() []authz.DynamicRolePolicy {
	name, ns, ok := getService(L.config)
	if !ok {
		return []authz.DynamicRolePolicy{}
	}

	prs := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{""},
			Resources:     []string{"endpoints"},
			Verbs:         []string{"get", "watch", "list"},
			ResourceNames: []string{name},
		},
	}

	return []authz.DynamicRolePolicy{
		{
			Namespaces: []string{ns},
			Rules:      prs,
		},
	}
}

func getService(config map[any]any) (name, namespace string, ok bool) {
	// key path: "resolver.k8s.service"
	service := ""
	if resolver, ok := config["resolver"].(map[any]any); ok {
		if k8s, ok := resolver["k8s"].(map[any]any); ok {
			service = k8s["service"].(string)
		}
	}
	if service == "" {
		return "", "", false
	}
	parts := strings.Split(service, ".")
	if len(parts) == 1 {
		// If there is no namespace, return an empty string "".
		// This is considered safe for subsequent ClusterRole creation logic.
		return parts[0], "", true
	}

	return parts[0], parts[1], true
}

func init() {
	parser.AuthzRegister(parser.ComponentTypeExporter, "loadbalancing", NewLBExporterParser)
}
