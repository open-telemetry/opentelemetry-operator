// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
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
					SourceLabels:         model.LabelNames{"i"},
					Action:               "replace",
					Separator:            ";",
					Regex:                relabel.MustNewRegexp("(.*)"),
					Replacement:          "$1",
					TargetLabel:          "foo",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"i"},
					Regex:                relabel.MustNewRegexp("(.*)"),
					Separator:            ";",
					Action:               "keep",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"i"},
					Regex:                relabel.MustNewRegexp("bad.*match"),
					Action:               "drop",
					Separator:            ";",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"label_not_present"},
					Regex:                relabel.MustNewRegexp("(.*)"),
					Separator:            ";",
					Action:               "keep",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: false,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"i"},
					Regex:                relabel.MustNewRegexp("(.*)"),
					Separator:            ";",
					Action:               "drop",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"collector"},
					Regex:                relabel.MustNewRegexp("(collector.*)"),
					Separator:            ";",
					Action:               "drop",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"i"},
					Regex:                relabel.MustNewRegexp("bad.*match"),
					Separator:            ";",
					Action:               "keep",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
				},
			},
			isDrop: true,
		},
		{
			cfg: []*relabel.Config{
				{
					SourceLabels:         model.LabelNames{"collector"},
					Regex:                relabel.MustNewRegexp("collectors-n"),
					Separator:            ";",
					Action:               "keep",
					Replacement:          "$1",
					NameValidationScheme: model.UTF8Validation,
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
		label := labels.New(
			labels.Label{Name: "collector", Value: collector},
			labels.Label{Name: "i", Value: strconv.Itoa(i)},
			labels.Label{Name: "total", Value: strconv.Itoa(n + startingIndex)},
			labels.Label{Name: model.MetaLabelPrefix + strconv.Itoa(i), Value: strconv.Itoa(i)},
			labels.Label{Name: model.AddressLabel, Value: "address_value"},
			// These labels are typically required for correct scraping behavior and are expected to be retained after relabeling.:
			//   - job
			//   - __scrape_interval__
			//   - __scrape_timeout__
			//   - __scheme__
			//   - __metrics_path__
			// Prometheus adds these labels by default. Removing them via relabel_configs is considered invalid and is therefore ignored.
			// For details, see:
			// https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L429
			labels.Label{Name: model.JobLabel, Value: jobName},
			labels.Label{Name: model.ScrapeIntervalLabel, Value: "10s"},
			labels.Label{Name: model.ScrapeTimeoutLabel, Value: "10s"},
			labels.Label{Name: model.SchemeLabel, Value: "http"},
			labels.Label{Name: model.MetricsPathLabel, Value: "/metrics" + strconv.Itoa(i)},

			// Prometheus will automatically add the "instance" label if it is not present.
			labels.Label{Name: model.InstanceLabel, Value: "address_value"},
		)
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
	assert.Equal(t, expectedTargetMap, remainingItems)

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

	for k, cfg := range relabelCfg {
		relabelCfg[k] = addNoShardingConfig(cfg)
	}
	actual := allocatorPrehook.GetConfig()
	assert.Equal(t, relabelCfg, actual)
}

func TestDistinctTarget(t *testing.T) {
	allocatorPrehook := New("relabel-config", logger)
	assert.NotNil(t, allocatorPrehook)

	targets, _, expectedTarget, relabelCfg, err := makeNNewTargets(relabelConfigs, 10, defaultNumCollectors, defaultStartIndex)
	assert.NoError(t, err)

	duplicatedTargets := make([]*target.Item, 0, 2*len(targets))
	for _, item := range targets {
		builder := labels.NewBuilder(item.Labels)
		builder.Set(checkDistinctConfigLabel, "check-distinct-label-value")
		ls := builder.Labels()

		duplItem := target.NewItem(item.JobName, item.TargetURL, ls, item.CollectorName)
		duplicatedTargets = append(duplicatedTargets, duplItem)
	}
	// Append original targets after duplicated ones to preserve original labels after deduplication.
	duplicatedTargets = append(duplicatedTargets, targets...)

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
	assert.Equal(t, promTargetMap, expectedTargetMap)

	// The deduplicated result after otel-allocator processing.
	allocatorPrehook.SetConfig(relabelCfg)
	remainingItems := allocatorPrehook.Apply(duplicatedTargets)
	remainingItemsMap := make(map[target.ItemHash]*target.Item)
	for _, item := range remainingItems {
		remainingItemsMap[item.Hash()] = item
	}

	assert.Len(t, remainingItemsMap, len(expectedTargetMap))
	assert.Equal(t, remainingItemsMap, expectedTargetMap)
}

func MakeTargetFromProm(rCfgs []*relabel.Config, rawTarget *target.Item) (*target.Item, error) {
	lb := labels.NewBuilder(rawTarget.Labels)
	cfg := &config.ScrapeConfig{
		RelabelConfigs: rCfgs,
	}
	lset, _, err := PopulateLabels(lb, cfg)
	if err != nil {
		return nil, err
	}
	// If the lset is empty after relabeling, Prometheus drops the target.
	if lset.IsEmpty() {
		return nil, nil
	}

	// Compute the hash from the builder, skipping meta labels
	hash := target.HashFromBuilder(lb, rawTarget.JobName)
	newTarget := target.NewItem(
		rawTarget.JobName,
		rawTarget.TargetURL,
		rawTarget.Labels,
		rawTarget.CollectorName,
		target.WithHash(hash),
	)
	return newTarget, nil
}

// PopulateLabels is Copied from prometheus/scrape/target.go.
// Reason: "github.com/prometheus/common@0.65.0" and "github.com/prometheus/scrape@0.301.0" are incompatible (undefined: promslog.AllowedFormat).
func PopulateLabels(lb *labels.Builder, cfg *config.ScrapeConfig) (res, orig labels.Labels, err error) {
	// Copy labels into the labelset for the target if they are not set already.
	scrapeLabels := []labels.Label{
		{Name: model.JobLabel, Value: cfg.JobName},
		{Name: model.ScrapeIntervalLabel, Value: cfg.ScrapeInterval.String()},
		{Name: model.ScrapeTimeoutLabel, Value: cfg.ScrapeTimeout.String()},
		{Name: model.MetricsPathLabel, Value: cfg.MetricsPath},
		{Name: model.SchemeLabel, Value: cfg.Scheme},
	}

	for _, l := range scrapeLabels {
		if lb.Get(l.Name) == "" {
			lb.Set(l.Name, l.Value)
		}
	}
	// Encode scrape query parameters as labels.
	for k, v := range cfg.Params {
		if name := model.ParamLabelPrefix + k; len(v) > 0 && lb.Get(name) == "" {
			lb.Set(name, v[0])
		}
	}

	preRelabelLabels := lb.Labels()
	keep := relabel.ProcessBuilder(lb, cfg.RelabelConfigs...)

	// Check if the target was dropped.
	if !keep {
		return labels.EmptyLabels(), preRelabelLabels, nil
	}
	if v := lb.Get(model.AddressLabel); v == "" {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("no address")
	}

	addr := lb.Get(model.AddressLabel)

	if err = config.CheckTargetAddress(model.LabelValue(addr)); err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}

	interval := lb.Get(model.ScrapeIntervalLabel)
	intervalDuration, err := model.ParseDuration(interval)
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("error parsing scrape interval: %w", err)
	}
	if time.Duration(intervalDuration) == 0 {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("scrape interval cannot be 0")
	}

	timeout := lb.Get(model.ScrapeTimeoutLabel)
	timeoutDuration, err := model.ParseDuration(timeout)
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("error parsing scrape timeout: %w", err)
	}
	if time.Duration(timeoutDuration) == 0 {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("scrape timeout cannot be 0")
	}

	if timeoutDuration > intervalDuration {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("scrape timeout cannot be greater than scrape interval (%q > %q)", timeout, interval)
	}

	// Meta labels are deleted after relabelling. Other internal labels propagate to
	// the target which decides whether they will be part of their label set.
	lb.Range(func(l labels.Label) {
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			lb.Del(l.Name)
		}
	})

	// Default the instance label to the target address.
	if v := lb.Get(model.InstanceLabel); v == "" {
		lb.Set(model.InstanceLabel, addr)
	}

	res = lb.Labels()
	err = res.Validate(func(l labels.Label) error {
		// Check label values are valid, drop the target if not.
		if !model.LabelValue(l.Value).IsValid() {
			return fmt.Errorf("invalid label value for %q: %q", l.Name, l.Value)
		}
		return nil
	})
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}
	return res, preRelabelLabels, nil
}
