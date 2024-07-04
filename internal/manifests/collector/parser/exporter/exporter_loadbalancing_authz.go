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
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

var _ parser.AuthzParser = &LBExporterParser{}

const (
	parserNameLoadBalancing = "__loadbalancing"
)

// LBExporterParser parses the configuration for OTLP receivers.
type LBExporterParser struct {
	config map[interface{}]interface{}
	logger logr.Logger
	name   string
}

// NewLBExporterParser builds a new parser loadbalancing processor.
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
	//TODO implement me
	panic("implement me")
}

func init() {
	// TODO fy
	parser.AuthzRegister(parser.ComponentTypeExporter, "loadbalancing", NewLBExporterParser)
}
