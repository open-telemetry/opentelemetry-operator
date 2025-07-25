// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"slices"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
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

func (tf *relabelConfigTargetFilter) Apply(targets []*target.Item) []*target.Item {
	numTargets := len(targets)

	// need to wait until relabelCfg is set
	if len(tf.relabelCfg) == 0 {
		return targets
	}

	writeIndex := 0
	for _, tItem := range targets {
		keepTarget := true
		lset := tItem.Labels
		for _, cfg := range tf.relabelCfg[tItem.JobName] {
			// These labels are typically required for correct scraping behavior and are expected to be retained after relabeling.:
			//   - job
			//   - __scrape_interval__
			//   - __scrape_timeout__
			//   - __scheme__
			//   - __metrics_path__
			// Prometheus adds these labels by default. Removing them via relabel_configs is considered invalid and is therefore ignored.
			// For details, see:
			// https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L429
			lset, keepTarget = relabel.Process(lset, cfg)
			if !keepTarget {
				break // inner loop
			}
		}

		if keepTarget {
			// Only if the key model.AddressLabel remains after relabeling is the value considered valid.
			// For detail, see https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L457
			address := lset.Get(model.AddressLabel)
			if len(address) == 0 {
				tf.log.V(2).Info("Dropping target because it has no __address__ label", "target", tItem)
				continue
			}
			targets[writeIndex] = target.NewItem(tItem.JobName, address, lset, tItem.CollectorName, target.WithReservedLabelMatching(tItem.Labels), target.WithFilterMetaLabels())
			writeIndex++
		}
	}

	targets = targets[:writeIndex]
	targets = slices.Clip(targets)
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
