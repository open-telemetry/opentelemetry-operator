// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"github.com/go-logr/logr"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

type Hook interface {
	Apply(map[string]*target.Item) map[string]*target.Item
	SetConfig(map[string][]*relabel.Config)
	GetConfig() map[string][]*relabel.Config
}

type HookProvider func(log logr.Logger) Hook

var (
	registry = map[string]HookProvider{
		relabelConfigTargetFilterName: newRelabelConfigTargetFilter,
	}
)

func New(name string, log logr.Logger) Hook {
	if p, ok := registry[name]; ok {
		return p(log.WithName("Prehook").WithName(name))
	}

	log.Info("Unrecognized filter strategy; filtering disabled")
	return nil
}
