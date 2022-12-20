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

package prehook

import (
	"errors"

	"github.com/go-logr/logr"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const (
	relabelConfigTargetFilterName = "relabel-config"
)

type Hook interface {
	Apply(map[string]*target.Item) map[string]*target.Item
	SetConfig(map[string][]*relabel.Config)
	GetConfig() map[string][]*relabel.Config
}

type HookProvider func(log logr.Logger) Hook

var (
	registry = map[string]HookProvider{}
)

func New(name string, log logr.Logger) Hook {
	if p, ok := registry[name]; ok {
		return p(log.WithName("Prehook").WithName(name))
	}

	log.Info("Unrecognized filter strategy; filtering disabled")
	return nil
}

func Register(name string, provider HookProvider) error {
	if _, ok := registry[name]; ok {
		return errors.New("already registered")
	}
	registry[name] = provider
	return nil
}

func init() {
	err := Register(relabelConfigTargetFilterName, NewRelabelConfigTargetFilter)
	if err != nil {
		panic(err)
	}
}
