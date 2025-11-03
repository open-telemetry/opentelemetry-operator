// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"slices"

	"github.com/go-logr/logr"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const (
	relabelConfigTargetFilterName = "relabel-config"
	disableShardingLabelName      = "__tmp_disable_sharding"
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

func (tf *relabelConfigTargetFilter) Apply(targets []*target.Item) []*target.Item {
	numTargets := len(targets)

	// need to wait until relabelCfg is set
	if len(tf.relabelCfg) == 0 {
		return targets
	}

	writeIndex := 0
	for _, tItem := range targets {
		newLabels, keepTarget := relabel.Process(tItem.Labels, tf.relabelCfg[tItem.JobName]...)

		if keepTarget {
			targets[writeIndex] = target.NewItem(tItem.JobName, tItem.TargetURL, tItem.Labels, tItem.CollectorName, target.WithRelabeledLabels(newLabels))
			writeIndex++
		}
	}

	targets = targets[:writeIndex]
	targets = slices.Clip(targets)
	tf.log.V(1).Info("Filtering complete", "seen", numTargets, "kept", len(targets))
	return targets
}

func (tf *relabelConfigTargetFilter) SetConfig(cfgs map[string][]*relabel.Config) {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for key, val := range cfgs {
		relabelCfgCopy[key] = addNoShardingConfig(val)
	}

	tf.relabelCfg = relabelCfgCopy
}

// addNoShardingConfig adds a relabel config to disable sharding for the given job. This is needed because the.
func addNoShardingConfig(cfg []*relabel.Config) []*relabel.Config {
	noShardingRelabelConfig := relabel.DefaultRelabelConfig
	noShardingRelabelConfig.Replacement = "true" // the value doesn't matter, it just needs to be non-empty
	noShardingRelabelConfig.TargetLabel = disableShardingLabelName
	dropTmpLabelConfig := relabel.DefaultRelabelConfig
	dropTmpLabelConfig.Action = relabel.LabelDrop
	dropTmpLabelConfig.Regex = relabel.MustNewRegexp(disableShardingLabelName)
	output := append([]*relabel.Config{&noShardingRelabelConfig}, cfg...)
	return append(output, &dropTmpLabelConfig)
}

func (tf *relabelConfigTargetFilter) GetConfig() map[string][]*relabel.Config {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for k, v := range tf.relabelCfg {
		relabelCfgCopy[k] = v
	}
	return relabelCfgCopy
}
