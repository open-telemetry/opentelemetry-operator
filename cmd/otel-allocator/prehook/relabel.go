// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"github.com/go-logr/logr"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const (
	relabelConfigTargetFilterName = "relabel-config"
)

type relabelConfigTargetFilter struct {
	log        logr.Logger
	relabelCfg map[string][]*relabel.Config
}

func newRelabelConfigTargetFilter(log logr.Logger) Hook {
	return &relabelConfigTargetFilter{
		log:        log,
		relabelCfg: make(map[string][]*relabel.Config),
	}
}

func (tf *relabelConfigTargetFilter) Apply(targets map[string]*target.Item) map[string]*target.Item {
	numTargets := len(targets)

	// need to wait until relabelCfg is set
	if len(tf.relabelCfg) == 0 {
		return targets
	}

	// Note: jobNameKey != tItem.JobName (jobNameKey is hashed)
	for jobNameKey, tItem := range targets {
		var keepTarget bool
		lset := tItem.Labels
		for _, cfg := range tf.relabelCfg[tItem.JobName] {
			lset, keepTarget = relabel.Process(lset, cfg)
			if !keepTarget {
				delete(targets, jobNameKey)
				break // inner loop
			}
		}
	}

	tf.log.V(2).Info("Filtering complete", "seen", numTargets, "kept", len(targets))
	return targets
}

func (tf *relabelConfigTargetFilter) SetConfig(cfgs map[string][]*relabel.Config) {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for key, val := range cfgs {
		relabelCfgCopy[key] = tf.replaceRelabelConfig(val)
	}

	tf.relabelCfg = relabelCfgCopy
}

// See this thread [https://github.com/open-telemetry/opentelemetry-operator/pull/1124/files#r983145795]
// for why SHARD == 0 is a necessary substitution. Otherwise the keep action that uses this env variable,
// would not match the regex and all targets end up dropped. Also note, $(SHARD) will always be 0 and it
// does not make sense to read from the environment because it is never set in the allocator.
func (tf *relabelConfigTargetFilter) replaceRelabelConfig(cfg []*relabel.Config) []*relabel.Config {
	for i := range cfg {
		str := cfg[i].Regex.String()
		if str == "$(SHARD)" {
			cfg[i].Regex = relabel.MustNewRegexp("0")
		}
	}

	return cfg
}

func (tf *relabelConfigTargetFilter) GetConfig() map[string][]*relabel.Config {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for k, v := range tf.relabelCfg {
		relabelCfgCopy[k] = v
	}
	return relabelCfgCopy
}
