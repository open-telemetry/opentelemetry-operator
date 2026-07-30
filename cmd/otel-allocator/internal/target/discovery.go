// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"context"
	"hash"
	"hash/fnv"
	"slices"
	"sync"
	"time"

	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap/zapcore"

	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

const labelBuilderPreallocSize = 100

// RelabelConfigFilterStrategy is the filter strategy that drops targets while they are being
// created, based on the scrape config's relabel_configs. It's the only filtering strategy
// currently supported; any other value disables target filtering.
const RelabelConfigFilterStrategy = "relabel-config"

type Discoverer struct {
	log                         logr.Logger
	manager                     discoveryManager
	close                       chan struct{}
	mtxScrape                   sync.Mutex // Guards the fields below.
	configsMap                  map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig
	relabelCfg                  map[string][]*relabel.Config
	filterRelabelConfig         bool
	scrapeConfigsHash           hash.Hash
	scrapeConfigsUpdater        scrapeConfigsUpdater
	targetSets                  map[string][]*targetgroup.Group
	triggerReload               chan struct{}
	processTargetsCallBack      func(targets []*Item)
	targetsDiscovered           metric.Float64Gauge
	processTargetsDuration      metric.Float64Histogram
	processTargetGroupsDuration metric.Float64Histogram
	reloadInterval              time.Duration
}

// DiscovererOption configures optional Discoverer behavior.
type DiscovererOption func(*Discoverer)

// WithReloadInterval sets how often the discoverer coalesces and applies target
// updates from service discovery. It defaults to defaultReloadInterval; tests can
// set a small value to avoid waiting on the debounce.
func WithReloadInterval(d time.Duration) DiscovererOption {
	return func(disc *Discoverer) { disc.reloadInterval = d }
}

const defaultReloadInterval = 5 * time.Second

type scrapeConfigsUpdater interface {
	UpdateScrapeConfigResponse(map[string]*promconfig.ScrapeConfig) error
}

// discoveryManager is the subset of *discovery.Manager the Discoverer depends on, so
// tests that inject target sets directly (via UpdateTsets) can supply a fake instead
// of running real service discovery.
type discoveryManager interface {
	ApplyConfig(cfg map[string]discovery.Configs) error
	SyncCh() <-chan map[string][]*targetgroup.Group
}

func NewDiscoverer(
	log logr.Logger,
	manager discoveryManager,
	filterStrategy string,
	scrapeConfigsUpdater scrapeConfigsUpdater,
	setTargets func(targets []*Item),
	opts ...DiscovererOption,
) (*Discoverer, error) {
	meter := otel.GetMeterProvider().Meter("targetallocator")
	targetsDiscovered, err := meter.Float64Gauge("opentelemetry_allocator_targets", metric.WithDescription("Number of targets discovered."))
	if err != nil {
		return nil, err
	}
	processTargetsDuration, err := meter.Float64Histogram("opentelemetry_allocator_process_targets_duration_seconds",
		metric.WithDescription("Duration of processing targets."), metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120))
	if err != nil {
		return nil, err
	}
	processTargetGroupsDuration, err := meter.Float64Histogram("opentelemetry_allocator_process_target_groups_duration_seconds",
		metric.WithDescription("Duration of processing target groups."), metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120))
	if err != nil {
		return nil, err
	}
	d := &Discoverer{
		log:                         log,
		manager:                     manager,
		close:                       make(chan struct{}),
		triggerReload:               make(chan struct{}, 1),
		configsMap:                  make(map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig),
		relabelCfg:                  make(map[string][]*relabel.Config),
		filterRelabelConfig:         filterStrategy == RelabelConfigFilterStrategy,
		scrapeConfigsHash:           nil, // we want the first update to succeed even if the config is empty
		scrapeConfigsUpdater:        scrapeConfigsUpdater,
		processTargetsCallBack:      setTargets,
		targetsDiscovered:           targetsDiscovered,
		processTargetsDuration:      processTargetsDuration,
		processTargetGroupsDuration: processTargetGroupsDuration,
		reloadInterval:              defaultReloadInterval,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d, nil
}

func (m *Discoverer) ApplyConfig(source allocatorWatcher.EventSource, scrapeConfigs []*promconfig.ScrapeConfig) error {
	m.configsMap[source] = scrapeConfigs
	jobToScrapeConfig := make(map[string]*promconfig.ScrapeConfig)

	discoveryCfg := make(map[string]discovery.Configs)
	relabelCfg := make(map[string][]*relabel.Config)

	for _, configs := range m.configsMap {
		for _, scrapeConfig := range configs {
			jobToScrapeConfig[scrapeConfig.JobName] = scrapeConfig
			discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
			// When the relabel-config filter strategy is enabled, relabeling is applied as targets
			// are created (see processTargetGroups). We add the no-sharding config here so it's
			// accounted for in the same place as the user's configs. When filtering is disabled,
			// we leave relabelCfg empty so no relabeling/filtering is done during discovery.
			if m.filterRelabelConfig {
				relabelCfg[scrapeConfig.JobName] = addNoShardingConfig(scrapeConfig.RelabelConfigs)
			}
		}
	}

	hash, err := getScrapeConfigHash(jobToScrapeConfig)
	if err != nil {
		return err
	}
	// If the hash has changed, updated stored hash and send the new config.
	// Otherwise, skip updating scrape configs.
	if m.scrapeConfigsUpdater != nil && m.scrapeConfigsHash != hash {
		err := m.scrapeConfigsUpdater.UpdateScrapeConfigResponse(jobToScrapeConfig)
		if err != nil {
			return err
		}

		m.scrapeConfigsHash = hash
	}

	m.mtxScrape.Lock()
	m.relabelCfg = relabelCfg
	m.mtxScrape.Unlock()

	return m.manager.ApplyConfig(discoveryCfg)
}

func (m *Discoverer) Run() error {
	err := m.run(m.manager.SyncCh())
	if err != nil {
		m.log.Error(err, "Service Discovery watch event failed")
		return err
	}
	<-m.close
	m.log.Info("Service Discovery watch event stopped: discovery manager closed")
	return nil
}

// UpdateTsets updates the target sets to be scraped.
func (m *Discoverer) UpdateTsets(tsets map[string][]*targetgroup.Group) {
	m.mtxScrape.Lock()
	m.targetSets = tsets
	m.mtxScrape.Unlock()
}

// reloader triggers a reload of the scrape configs at regular intervals.
// The time between reloads is m.reloadInterval, to avoid overloading the system
// with too many reloads, because some service discovery mechanisms can be quite chatty.
func (m *Discoverer) reloader() {
	ticker := time.NewTicker(m.reloadInterval)

	defer ticker.Stop()

	for {
		select {
		case <-m.close:
			return
		case <-ticker.C:
			select {
			case <-m.triggerReload:
				m.Reload()
			case <-m.close:
				return
			}
		}
	}
}

// Reload triggers a reload of the scrape configs.
// This will process the target groups and update the targets concurrently.
func (m *Discoverer) Reload() {
	m.mtxScrape.Lock()
	var wg sync.WaitGroup
	begin := time.Now()
	defer func() {
		m.processTargetsDuration.Record(context.Background(), time.Since(begin).Seconds())
	}()

	// Process each job's target groups in parallel, collecting the targets kept per job.
	// Relabeling, applied while creating the targets, can drop some of them, so the number of
	// targets per job isn't known up front. Each job writes into its own slice and we
	// concatenate them once all jobs are done.
	jobResults := make([][]*Item, len(m.targetSets))
	jobIndex := 0
	for jobName, groups := range m.targetSets {
		relabelCfg := m.relabelCfg[jobName]
		wg.Add(1)
		// Run the sync in parallel as these take a while and at high load can't catch up.
		go func(idx int, jobName string, groups []*targetgroup.Group, relabelCfg []*relabel.Config) {
			defer wg.Done()
			jobResults[idx] = m.processTargetGroups(jobName, groups, relabelCfg)
		}(jobIndex, jobName, groups, relabelCfg)
		jobIndex++
	}
	m.mtxScrape.Unlock()
	wg.Wait()

	targetCount := 0
	for _, result := range jobResults {
		targetCount += len(result)
	}
	targets := make([]*Item, 0, targetCount)
	for _, result := range jobResults {
		targets = append(targets, result...)
	}
	m.processTargetsCallBack(targets)
}

// processTargetGroups processes the target groups for a single job and returns the targets to be
// scraped. The job's relabel configuration is applied as each target is created: targets dropped
// by relabeling are excluded from the result, and for the targets that are kept the hash is
// computed from the relabeled labels while the builder is still available, avoiding a later
// recomputation.
func (m *Discoverer) processTargetGroups(jobName string, groups []*targetgroup.Group, relabelCfg []*relabel.Config) []*Item {
	// the builder for group labels
	groupBuilder := labels.NewScratchBuilder(labelBuilderPreallocSize)

	// a slice for sorting target label names, we allocate it here to avoid doing it in the hot loop
	targetLabelNames := make([]model.LabelName, 0, labelBuilderPreallocSize)

	// the builder used to apply relabeling to each target
	relabelBuilder := labels.NewBuilder(labels.EmptyLabels())

	begin := time.Now()
	defer func() {
		m.processTargetGroupsDuration.Record(context.Background(), time.Since(begin).Seconds(), metric.WithAttributes(attribute.String("job.name", jobName)))
	}()

	// preallocate assuming no targets are dropped by relabeling
	targetCount := 0
	for _, tg := range groups {
		targetCount += len(tg.Targets)
	}
	targets := make([]*Item, 0, targetCount)

	var count float64
	// Reusable slice for the sorted group labels, copied out of the builder once per group so the
	// per-target merge below can reuse (and overwrite) the builder without losing the group labels.
	groupSlice := make([]labels.Label, 0, labelBuilderPreallocSize)
	var groupLabels labels.Labels

	for _, tg := range groups {
		groupBuilder.Reset()
		for ln, lv := range tg.Labels {
			groupBuilder.Add(string(ln), string(lv))
		}
		groupBuilder.Sort()
		// Overwrite reuses the builder's internal buffer (no allocation after the first group).
		groupBuilder.Overwrite(&groupLabels)
		groupSlice = groupSlice[:0]
		groupLabels.Range(func(l labels.Label) {
			groupSlice = append(groupSlice, l)
		})

		for _, t := range tg.Targets {
			count++
			// Merge the sorted group labels with the target's labels into a single, globally sorted
			// label set. Reusing groupBuilder is safe because groupSlice holds an independent copy of
			// the group labels. The order matters: downstream consumers (and the conformance suite)
			// rely on Item.Labels honoring Prometheus' sorted labels.Labels invariant.
			targetBuilder := &groupBuilder
			targetBuilder.Reset()
			targetLabelNames = targetLabelNames[:0]
			mergeLabels(targetBuilder, groupSlice, t, targetLabelNames)
			itemLabels := targetBuilder.Labels()

			// Apply relabeling, then compute the target hash from the (possibly relabeled) labels
			// while we still have the builder, skipping meta labels. Targets dropped by relabeling
			// are excluded. We hash from the builder in both cases so the hash is computed once,
			// here, rather than lazily later.
			relabelBuilder.Reset(itemLabels)
			if len(relabelCfg) > 0 {
				if keepTarget := relabel.ProcessBuilder(relabelBuilder, relabelCfg...); !keepTarget {
					continue
				}
			}
			hash := HashFromBuilder(relabelBuilder, jobName)

			targets = append(targets, NewItem(jobName, string(t[model.AddressLabel]), itemLabels, "", hash))
		}
	}
	m.targetsDiscovered.Record(context.Background(), count, metric.WithAttributes(attribute.String("job.name", jobName)))
	return targets
}

const disableShardingLabelName = "__tmp_disable_sharding"

// addNoShardingConfig adds a relabel config to disable sharding for the given job. This is needed because the scrape
// configs generated by prometheus-operator by default depend on a `SHARD` environment variable, even in non-sharded
// Prometheus deployments. We don't want to set this variable on all collector deployments, so we instead disable
// the feature.
func addNoShardingConfig(cfg []*relabel.Config) []*relabel.Config {
	noShardingRelabelConfig := relabel.DefaultRelabelConfig
	noShardingRelabelConfig.Replacement = "true" // the value doesn't matter, it just needs to be non-empty
	noShardingRelabelConfig.TargetLabel = disableShardingLabelName

	// we need to drop the temporary label at the end
	dropTmpLabelConfig := relabel.DefaultRelabelConfig
	dropTmpLabelConfig.Action = relabel.LabelDrop
	dropTmpLabelConfig.Regex = relabel.MustNewRegexp(disableShardingLabelName)
	output := append([]*relabel.Config{&noShardingRelabelConfig}, cfg...)
	return append(output, &dropTmpLabelConfig)
}

// mergeLabels merges sorted group labels with target labels into the builder.
// Target labels override group labels on name collision.
func mergeLabels(builder *labels.ScratchBuilder, groupSlice []labels.Label, targetLabels model.LabelSet, targetLabelNamesBuf []model.LabelName) {
	for ln := range targetLabels {
		targetLabelNamesBuf = append(targetLabelNamesBuf, ln)
	}
	slices.Sort(targetLabelNamesBuf)

	gi, ti := 0, 0
	for gi < len(groupSlice) && ti < len(targetLabelNamesBuf) {
		gn := groupSlice[gi].Name
		tn := string(targetLabelNamesBuf[ti])
		switch {
		case gn < tn:
			builder.Add(gn, groupSlice[gi].Value)
			gi++
		case gn > tn:
			builder.Add(tn, string(targetLabels[targetLabelNamesBuf[ti]]))
			ti++
		default: // target label overrides group label
			builder.Add(tn, string(targetLabels[targetLabelNamesBuf[ti]]))
			gi++
			ti++
		}
	}
	for ; gi < len(groupSlice); gi++ {
		builder.Add(groupSlice[gi].Name, groupSlice[gi].Value)
	}
	for ; ti < len(targetLabelNamesBuf); ti++ {
		builder.Add(string(targetLabelNamesBuf[ti]), string(targetLabels[targetLabelNamesBuf[ti]]))
	}
}

// Run receives and saves target set updates and triggers the scraping loops reloading.
// Reloading happens in the background so that it doesn't block receiving targets updates.
func (m *Discoverer) run(tsets <-chan map[string][]*targetgroup.Group) error {
	go m.reloader()
	for {
		select {
		case ts := <-tsets:
			m.log.V(int(zapcore.DebugLevel)).Info("Service Discovery watch event received", "targets groups", len(ts))
			m.UpdateTsets(ts)

			select {
			case m.triggerReload <- struct{}{}:
			default:
			}

		case <-m.close:
			m.log.Info("Service Discovery watch event stopped: discovery manager closed")
			return nil
		}
	}
}

func (m *Discoverer) Close() {
	close(m.close)
}

// Calculate a hash for a scrape config map.
// This is done by marshaling to YAML because it's the most straightforward and doesn't run into problems with unexported fields.
func getScrapeConfigHash(jobToScrapeConfig map[string]*promconfig.ScrapeConfig) (hash.Hash64, error) {
	hash := fnv.New64()
	yamlEncoder := go_yaml.NewEncoder(hash)
	for jobName, scrapeConfig := range jobToScrapeConfig {
		_, err := hash.Write([]byte(jobName))
		if err != nil {
			return nil, err
		}
		err = yamlEncoder.Encode(scrapeConfig)
		if err != nil {
			return nil, err
		}
	}
	yamlEncoder.Close()
	return hash, nil
}
