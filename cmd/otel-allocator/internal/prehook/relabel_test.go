// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	logger               = logf.Log.WithName("unit-tests")
	defaultNumTargets    = 100
	defaultNumCollectors = 3
	defaultStartIndex    = 0

	checkDistinctConfigLabel = "check-distinct-label-key"

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

	CheckDistinctConfig = relabel.Config{
		Regex:  relabel.MustNewRegexp(checkDistinctConfigLabel),
		Action: "labeldrop",
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

func makeNNewTargets(rCfgs []relabelConfigObj, n int, numCollectors int, startingIndex int) ([]*target.Item, int, []*target.Item, map[string][]*relabel.Config, error) {
	toReturn := []*target.Item{}
	expected := []*target.Item{}
	numItemsRemaining := n
	relabelConfig := make(map[string][]*relabel.Config)
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		jobName := fmt.Sprintf("test-job-%d", i)
		label := labels.Labels{
			{Name: "collector", Value: collector},
			{Name: "i", Value: strconv.Itoa(i)},
			{Name: "total", Value: strconv.Itoa(n + startingIndex)},
			{Name: model.MetaLabelPrefix + strconv.Itoa(i), Value: strconv.Itoa(i)},
			{Name: model.AddressLabel, Value: "address_value"},
			// These labels are typically required for correct scraping behavior and are expected to be retained after relabeling.:
			//   - job
			//   - __scrape_interval__
			//   - __scrape_timeout__
			//   - __scheme__
			//   - __metrics_path__
			// Prometheus adds these labels by default. Removing them via relabel_configs is considered invalid and is therefore ignored.
			// For details, see:
			// https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L429
			{Name: model.JobLabel, Value: jobName},
			{Name: model.ScrapeIntervalLabel, Value: "10s"},
			{Name: model.ScrapeTimeoutLabel, Value: "10s"},
			{Name: model.SchemeLabel, Value: "http"},
			// Make sure the relabeled targets are unique to verify target deduplication.
			// For details, see function TestDistinctTarget.
			{Name: model.MetricsPathLabel, Value: "/metrics" + strconv.Itoa(i)},

			// Prometheus will automatically add the "instance" label if it is not present.
			{Name: model.InstanceLabel, Value: "address_value"},
		}
		rawTarget := target.NewItem(jobName, "test-url", label, collector)
		// add a single replace, drop, or keep action as relabel_config for targets
		var index int
		ind, _ := rand.Int(rand.Reader, big.NewInt(int64(len(relabelConfigs))))

		index = int(ind.Int64())

		relabelConfig[jobName] = rCfgs[index].cfg

		if relabelConfigs[index].isDrop {
			numItemsRemaining--
		} else {
			newTarget, err := MakeTargetFromProm(relabelConfig[jobName], rawTarget)
			if err != nil || newTarget == nil {
				return nil, 0, nil, nil, fmt.Errorf("failed to create target from relabel config: %w", err)
			}
			expected = append(expected, newTarget)
		}
		toReturn = append(toReturn, rawTarget)
	}
	return toReturn, numItemsRemaining, expected, relabelConfig, nil
}

func TestApply(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, numRemaining, expectedTargetMap, relabelCfg, err := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

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
	targets, numRemaining, expectedTargetMap, relabelCfg, err := makeNNewTargets(hashRelabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)
	assert.Len(t, remainingItems, numRemaining)
	assert.Equal(t, remainingItems, expectedTargetMap)
}

func TestApplyEmptyRelabelCfg(t *testing.T) {

	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, _, _, _, err := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

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

	_, _, _, relabelCfg, err := makeNNewTargets(relabelConfigs, defaultNumTargets, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

	allocatorPrehook.SetConfig(relabelCfg)
	assert.Equal(t, relabelCfg, allocatorPrehook.GetConfig())
}

func TestRemoveRelabelConfigs(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, _, _, relabelCfg, err := makeNNewTargets(relabelConfigs, 10, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

	// The original target after being processed by Prometheus relabeling.
	expectedTarget1 := make([]*target.Item, 0, len(targets))
	for _, item := range targets {
		tfp, err := MakeTargetFromProm(relabelCfg[item.JobName], item)
		assert.NoError(t, err)
		// If the target is dropped by Prometheus, it will be nil.
		if tfp != nil {
			expectedTarget1 = append(expectedTarget1, tfp)
		}
	}

	// The targets after being relabeled by otel-allocator.
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(targets)

	// The target processed by Prometheus relabeling after being handled by otel-allocator.
	expectedTarget2 := make([]*target.Item, 0, len(remainingItems))
	for _, item := range remainingItems {
		tfp, err := MakeTargetFromProm(nil, item)
		assert.NoError(t, err)

		expectedTarget2 = append(expectedTarget2, tfp)
	}

	assert.Len(t, expectedTarget1, len(expectedTarget2))
	assert.Equal(t, expectedTarget1, expectedTarget2)
}

func TestDistinctTarget(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, _, expectedTarget, relabelCfg, err := makeNNewTargets(relabelConfigs, 10, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

	duplicatedTargets := make([]*target.Item, 0, 2*len(targets))
	duplicatedTargets = append(duplicatedTargets, targets...)
	for _, item := range targets {
		ls := item.Labels.Copy()
		ls = append(ls, labels.Label{
			Name:  checkDistinctConfigLabel,
			Value: "check-distinct-label-value",
		})

		duplItem := target.NewItem(item.JobName, item.TargetURL, ls, item.CollectorName)
		duplicatedTargets = append(duplicatedTargets, duplItem)
	}

	for k, cfg := range relabelCfg {
		cfg = append(cfg, &CheckDistinctConfig)
		relabelCfg[k] = cfg
	}

	// The expected result after deduplication.
	expectedTargetMap := make(map[target.ItemHash]*target.Item)
	for _, item := range expectedTarget {
		expectedTargetMap[item.Hash()] = item
	}

	// The deduplicated result after Prometheus relabeling.
	promTargetMap := make(map[target.ItemHash]*target.Item)
	for _, item := range targets {
		tfp, err := MakeTargetFromProm(relabelCfg[item.JobName], item)
		assert.NoError(t, err)
		// If the target is dropped by Prometheus, it will be nil.
		if tfp != nil {
			promTargetMap[tfp.Hash()] = tfp
		}
	}

	assert.Len(t, promTargetMap, len(expectedTargetMap))
	assert.True(t, CompareTargetsMap(promTargetMap, expectedTargetMap), "The Prometheus relabeled targets should match the expected target map")

	// The deduplicated result after otel-allocator processing.
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(duplicatedTargets)
	assert.Less(t, len(remainingItems), len(duplicatedTargets), "The remainingItems should be less than the duplicated targets")

	remainingItemsMap := make(map[target.ItemHash]*target.Item)
	for _, item := range remainingItems {
		remainingItemsMap[item.Hash()] = item
	}

	assert.Len(t, remainingItemsMap, len(expectedTargetMap))
	assert.True(t, CompareTargetsMap(remainingItemsMap, expectedTargetMap), "The remaining items should match the expected target map")
}
