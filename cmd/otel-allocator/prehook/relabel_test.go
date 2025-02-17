// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	logger               = logf.Log.WithName("unit-tests")
	defaultNumTargets    = 100
	defaultNumCollectors = 3
	defaultStartIndex    = 0

	relabelConfigs = []relabelConfigObj{
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"i"},
					Action:       "replace",
					Separator:    ";",
					Regex:        relabel.MustNewRegexp("(.*)"),
					Replacement:  "$1",
					TargetLabel:  "foo",
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"i"},
					Regex:        relabel.MustNewRegexp("(.*)"),
					Separator:    ";",
					Action:       "keep",
					Replacement:  "$1",
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"i"},
					Regex:        relabel.MustNewRegexp("bad.*match"),
					Action:       "drop",
					Separator:    ";",
					Replacement:  "$1",
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"label_not_present"},
					Regex:        relabel.MustNewRegexp("(.*)"),
					Separator:    ";",
					Action:       "keep",
					Replacement:  "$1",
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"i"},
					Regex:        relabel.MustNewRegexp("(.*)"),
					Separator:    ";",
					Action:       "drop",
					Replacement:  "$1",
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"collector"},
					Regex:        relabel.MustNewRegexp("(collector.*)"),
					Separator:    ";",
					Action:       "drop",
					Replacement:  "$1",
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"i"},
					Regex:        relabel.MustNewRegexp("bad.*match"),
					Separator:    ";",
					Action:       "keep",
					Replacement:  "$1",
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels: model.LabelNames{"collector"},
					Regex:        relabel.MustNewRegexp("collectors-n"),
					Separator:    ";",
					Action:       "keep",
					Replacement:  "$1",
				},
			},
			isDrop: true,
		},
	}

	HashmodConfig = relabelConfigObj{
		cfg: []*relabel.Config{
			{
				SourceLabels: model.LabelNames{"i"},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Separator:    ";",
				Modulus:      1,
				TargetLabel:  "tmp-0",
				Action:       "hashmod",
				Replacement:  "$1",
			},

			{
				SourceLabels: model.LabelNames{"tmp-$(SHARD)"},
				Regex:        relabel.MustNewRegexp("$(SHARD)"),
				Separator:    ";",
				Action:       "keep",
				Replacement:  "$1",
			},
		},
		isDrop: false,
	}

	DefaultDropRelabelConfig = relabel.Config{
		SourceLabels: model.LabelNames{"i"},
		Regex:        relabel.MustNewRegexp("(.*)"),
		Action:       "drop",
	}
)

type relabelConfigObj struct {
	cfg    []*relabel.Config
	isDrop bool
}

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func makeNNewTargets(rCfgs []relabelConfigObj, n int, numCollectors int, startingIndex int) (map[string]*target.Item, int, map[string]*target.Item, map[string][]*relabel.Config) {
	toReturn := map[string]*target.Item{}
	expectedMap := make(map[string]*target.Item)
	numItemsRemaining := n
	relabelConfig := make(map[string][]*relabel.Config)
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := labels.Labels{
			{Name: "collector", Value: collector},
			{Name: "i", Value: strconv.Itoa(i)},
			{Name: "total", Value: strconv.Itoa(n + startingIndex)},
		}
		jobName := fmt.Sprintf("test-job-%d", i)
		newTarget := target.NewItem(jobName, "test-url", label, collector)
		// add a single replace, drop, or keep action as relabel_config for targets
		var index int
		ind, _ := rand.Int(rand.Reader, big.NewInt(int64(len(relabelConfigs))))

		index = int(ind.Int64())

		relabelConfig[jobName] = rCfgs[index].cfg

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
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, numRemaining, expectedTargetMap, relabelCfg := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
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

func TestApplyHashmodAction(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	hashRelabelConfigs := append(relabelConfigs, HashmodConfig)
	targets, numRemaining, expectedTargetMap, relabelCfg := makeNNewTargets(hashRelabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)
	assert.Len(t, remainingItems, numRemaining)
	assert.Equal(t, remainingItems, expectedTargetMap)
}

func TestApplyEmptyRelabelCfg(t *testing.T) {

	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, _, _, _ := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)

	relabelCfg := map[string][]*relabel.Config{}
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)
	// relabelCfg is empty so targets should be unfiltered
	assert.Len(t, remainingItems, len(targets))
	assert.Equal(t, remainingItems, targets)
}

func TestSetConfig(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	_, _, _, relabelCfg := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	allocatorPrehook.SetConfig(relabelCfg)
	assert.Equal(t, relabelCfg, allocatorPrehook.GetConfig())
}
