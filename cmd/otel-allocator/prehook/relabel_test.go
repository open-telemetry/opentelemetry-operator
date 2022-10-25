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
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	logger     = logf.Log.WithName("unit-tests")
	numTargets = 100

	relabelConfigs = []relabelConfigObj{
		{
			cfg: relabel.Config{
				Action:      "replace",
				Separator:   ";",
				Regex:       relabel.MustNewRegexp("(.*)"),
				Replacement: "$1",
			},
			isDrop: false,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"i"},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Action:       "keep",
			},
			isDrop: false,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"i"},
				Regex:        relabel.MustNewRegexp("bad.*match"),
				Action:       "drop",
			},
			isDrop: false,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"label_not_present"},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Action:       "keep",
			},
			isDrop: false,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"i"},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Action:       "drop",
			},
			isDrop: true,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"collector"},
				Regex:        relabel.MustNewRegexp("(collector.*)"),
				Action:       "drop",
			},
			isDrop: true,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"i"},
				Regex:        relabel.MustNewRegexp("bad.*match"),
				Action:       "keep",
			},
			isDrop: true,
		},
		{
			cfg: relabel.Config{
				SourceLabels: model.LabelNames{"collector"},
				Regex:        relabel.MustNewRegexp("collectors-n"),
				Action:       "keep",
			},
			isDrop: true,
		},
	}

	DefaultDropRelabelConfig = relabel.Config{
		SourceLabels: model.LabelNames{"i"},
		Regex:        relabel.MustNewRegexp("(.*)"),
		Action:       "drop",
	}
)

type relabelConfigObj struct {
	cfg    relabel.Config
	isDrop bool
}

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func makeNNewTargets(n int, numCollectors int, startingIndex int) (map[string]*target.TargetItem, int, map[string]*target.TargetItem, map[string][]*relabel.Config) {
	toReturn := map[string]*target.TargetItem{}
	expectedMap := make(map[string]*target.TargetItem)
	numItemsRemaining := n
	relabelConfig := make(map[string][]*relabel.Config)
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := model.LabelSet{
			"collector": model.LabelValue(collector),
			"i":         model.LabelValue(strconv.Itoa(i)),
			"total":     model.LabelValue(strconv.Itoa(n + startingIndex)),
		}
		jobName := fmt.Sprintf("test-job-%d", i)
		newTarget := target.NewTargetItem(jobName, "test-url", label, collector)
		// add a single replace, drop, or keep action as relabel_config for targets
		var index int
		ind, _ := rand.Int(rand.Reader, big.NewInt(int64(len(relabelConfigs))))

		index = int(ind.Int64())

		relabelConfig[jobName] = []*relabel.Config{
			&relabelConfigs[index].cfg,
		}

		targetKey := newTarget.Hash()
		if relabelConfigs[index].isDrop {
			numItemsRemaining--
		} else {
			expectedMap[targetKey] = newTarget
		}
		toReturn[targetKey] = newTarget
	}
	return toReturn, numItemsRemaining, expectedMap, relabelConfig
}

func TestApply(t *testing.T) {
	allocatorPrehook, err := New("relabel-config", logger)
	assert.Nil(t, err)

	targets, numRemaining, expectedTargetMap, relabelCfg := makeNNewTargets(numTargets, 3, 0)
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)
	assert.Len(t, remainingItems, numRemaining)
	assert.Equal(t, remainingItems, expectedTargetMap)

	// clear out relabelCfg to test with empty values
	for key := range relabelCfg {
		relabelCfg[key] = nil
	}

	// cfg = createMockConfig(relabelCfg)
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems = allocatorPrehook.Apply(targets)
	// relabelCfg is empty so targets should be unfiltered
	assert.Len(t, remainingItems, len(targets))
	assert.Equal(t, remainingItems, targets)
}

func TestApplyEmptyRelabelCfg(t *testing.T) {

	allocatorPrehook, err := New("relabel-config", logger)
	assert.Nil(t, err)

	targets, _, _, _ := makeNNewTargets(numTargets, 3, 0)

	relabelCfg := map[string][]*relabel.Config{}
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)
	// relabelCfg is empty so targets should be unfiltered
	assert.Len(t, remainingItems, len(targets))
	assert.Equal(t, remainingItems, targets)
}

func TestSetConfig(t *testing.T) {
	allocatorPrehook, err := New("relabel-config", logger)
	assert.Nil(t, err)

	_, _, _, relabelCfg := makeNNewTargets(numTargets, 3, 0)
	allocatorPrehook.SetConfig(relabelCfg)
	assert.Equal(t, relabelCfg, allocatorPrehook.GetConfig())
}
