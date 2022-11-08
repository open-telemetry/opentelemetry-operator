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
	"bytes"

	"github.com/go-logr/logr"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	yaml2 "gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

type RelabelConfigTargetFilter struct {
	log        logr.Logger
	relabelCfg map[string][]*relabel.Config
}

func NewRelabelConfigTargetFilter(log logr.Logger) Hook {
	return &RelabelConfigTargetFilter{
		log:        log,
		relabelCfg: make(map[string][]*relabel.Config),
	}
}

// helper function converts from model.LabelSet to []labels.Label.
func convertLabelToPromLabelSet(lbls model.LabelSet) []labels.Label {
	newLabels := make([]labels.Label, len(lbls))
	index := 0
	for k, v := range lbls {
		newLabels[index].Name = string(k)
		newLabels[index].Value = string(v)
		index++
	}
	return newLabels
}

func (tf *RelabelConfigTargetFilter) Apply(targets map[string]*target.Item) map[string]*target.Item {
	numTargets := len(targets)

	// need to wait until relabelCfg is set
	if len(tf.relabelCfg) == 0 {
		return targets
	}

	// Note: jobNameKey != tItem.JobName (jobNameKey is hashed)
	for jobNameKey, tItem := range targets {
		keepTarget := true
		lset := convertLabelToPromLabelSet(tItem.Label)
		for _, cfg := range tf.relabelCfg[tItem.JobName] {
			if new_lset := relabel.Process(lset, cfg); new_lset == nil {
				keepTarget = false
				break // inner loop
			} else {
				lset = new_lset
			}
		}

		if !keepTarget {
			delete(targets, jobNameKey)
		}
	}

	tf.log.V(2).Info("Filtering complete", "seen", numTargets, "kept", len(targets))
	return targets
}

func (tf *RelabelConfigTargetFilter) SetConfig(cfgs map[string][]*relabel.Config) {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for key, val := range cfgs {
		relabelCfgCopy[key] = val
	}

	out, err := yaml2.Marshal(relabelCfgCopy)
	if err != nil {
		tf.log.V(2).Info("Error Marshaling", "error", err)
		return
	}

	byteArr := replaceShard([]byte(out))
	err = yaml2.Unmarshal(byteArr, &relabelCfgCopy)
	if err != nil {
		tf.log.Info("Error Unmarshalling", "error", err)
		return
	}
	tf.relabelCfg = relabelCfgCopy
}

func (tf *RelabelConfigTargetFilter) GetConfig() map[string][]*relabel.Config {
	relabelCfgCopy := make(map[string][]*relabel.Config)
	for k, v := range tf.relabelCfg {
		relabelCfgCopy[k] = v
	}
	return relabelCfgCopy
}

func replaceShard(body []byte) []byte {
	// See this thread [https://github.com/open-telemetry/opentelemetry-operator/pull/1124/files#r983145795]
	// for why SHARD == 0 is a necessary substitution. Otherwise the keep action that uses this env variable,
	// would not match the regex and all targets end up dropped.
	shard := "0"
	return bytes.ReplaceAll(body, []byte("$(SHARD)"), []byte(shard))
}

func init() {
	err := Register(relabelConfigTargetFilterName, NewRelabelConfigTargetFilter)
	if err != nil {
		panic(err)
	}
}
