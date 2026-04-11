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

type Discoverer struct {
	log                         logr.Logger
	manager                     *discovery.Manager
	close                       chan struct{}
	mtxScrape                   sync.Mutex // Guards the fields below.
	configsMap                  map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig
	hook                        discoveryHook
	scrapeConfigsHash           hash.Hash
	scrapeConfigsUpdater        scrapeConfigsUpdater
	targetSets                  map[string][]*targetgroup.Group
	triggerReload               chan struct{}
	processTargetsCallBack      func(targets []*Item)
	targetsDiscovered           metric.Float64Gauge
	processTargetsDuration      metric.Float64Histogram
	processTargetGroupsDuration metric.Float64Histogram
}

type discoveryHook interface {
	SetConfig(map[string][]*relabel.Config)
}

type scrapeConfigsUpdater interface {
	UpdateScrapeConfigResponse(map[string]*promconfig.ScrapeConfig) error
}

func NewDiscoverer(log logr.Logger, manager *discovery.Manager, hook discoveryHook, scrapeConfigsUpdater scrapeConfigsUpdater, setTargets func(targets []*Item)) (*Discoverer, error) {
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
	return &Discoverer{
		log:                         log,
		manager:                     manager,
		close:                       make(chan struct{}),
		triggerReload:               make(chan struct{}, 1),
		configsMap:                  make(map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig),
		hook:                        hook,
		scrapeConfigsHash:           nil, // we want the first update to succeed even if the config is empty
		scrapeConfigsUpdater:        scrapeConfigsUpdater,
		processTargetsCallBack:      setTargets,
		targetsDiscovered:           targetsDiscovered,
		processTargetsDuration:      processTargetsDuration,
		processTargetGroupsDuration: processTargetGroupsDuration,
	}, nil
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
			relabelCfg[scrapeConfig.JobName] = scrapeConfig.RelabelConfigs
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

	if m.hook != nil {
		m.hook.SetConfig(relabelCfg)
	}
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
// The time between reloads is defined by reloadIntervalDuration to avoid overloading the system
// with too many reloads, because some service discovery mechanisms can be quite chatty.
func (m *Discoverer) reloader() {
	reloadIntervalDuration := model.Duration(5 * time.Second)
	ticker := time.NewTicker(time.Duration(reloadIntervalDuration))

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

	// count targets and preallocate
	targetCount := 0
	for _, groups := range m.targetSets {
		for _, group := range groups {
			targetCount += len(group.Targets)
		}
	}
	targets := make([]*Item, targetCount)

	targetsAssigned := 0
	for jobName, groups := range m.targetSets {
		wg.Add(1)
		// Run the sync in parallel as these take a while and at high load can't catch up.
		go func(jobName string, groups []*targetgroup.Group, intoTargets []*Item) {
			m.processTargetGroups(jobName, groups, intoTargets)
			wg.Done()
		}(jobName, groups, targets[targetsAssigned:])
		for _, group := range groups {
			targetsAssigned += len(group.Targets)
		}
	}
	m.mtxScrape.Unlock()
	wg.Wait()
	m.processTargetsCallBack(targets)
}

// processTargetGroups processes the target groups and returns a map of targets.
func (m *Discoverer) processTargetGroups(jobName string, groups []*targetgroup.Group, intoTargets []*Item) {
	// the builder for group labels
	groupBuilder := labels.NewScratchBuilder(labelBuilderPreallocSize)

	// a slice for sorting target label names, we allocate it here to avoid doing it in the hot loop
	targetLabelNames := make([]string, 0, labelBuilderPreallocSize)

	begin := time.Now()
	defer func() {
		m.processTargetGroupsDuration.Record(context.Background(), time.Since(begin).Seconds(), metric.WithAttributes(attribute.String("job.name", jobName)))
	}()
	var count float64
	index := 0
	// Reusable sorted group labels for indexed merge access.
	groupSlice := make([]labels.Label, 0, labelBuilderPreallocSize)
	// Overwrite target for group labels — reuses internal buffer across groups.
	var groupLabels labels.Labels

	for _, tg := range groups {
		groupBuilder.Reset()
		for ln, lv := range tg.Labels {
			groupBuilder.Add(string(ln), string(lv))
		}
		groupBuilder.Sort()
		// Overwrite reuses the builder's internal buffer (no allocation after first group).
		groupBuilder.Overwrite(&groupLabels)
		groupSlice = groupSlice[:0]
		groupLabels.Range(func(l labels.Label) {
			groupSlice = append(groupSlice, l)
		})

		// Pointer alias: reuse groupBuilder for per-target merged labels.
		targetBuilder := &groupBuilder

		for _, t := range tg.Targets {
			count++
			targetBuilder.Reset()

			// Sort target label names (typically very few: __address__, __metrics_path__).
			targetLabelNames = targetLabelNames[:0]
			for ln := range t {
				targetLabelNames = append(targetLabelNames, string(ln))
			}
			slices.Sort(targetLabelNames)

			// Merge two sorted sequences (group labels + target labels) to produce
			// globally sorted output without a final Sort() call. This preserves the
			// performance spirit of PR #4587's struct-copy optimization while ensuring
			// labels.Labels.Get() (binary search) works correctly. The previous
			// struct-copy approach produced two independently sorted sublists that
			// broke Get("__address__") when group labels sorted after it alphabetically.
			gi, ti := 0, 0
			for gi < len(groupSlice) && ti < len(targetLabelNames) {
				gn := groupSlice[gi].Name
				tn := targetLabelNames[ti]
				switch {
				case gn < tn:
					targetBuilder.Add(gn, groupSlice[gi].Value)
					gi++
				case gn > tn:
					targetBuilder.Add(tn, string(t[model.LabelName(tn)]))
					ti++
				default: // target label overrides group label
					targetBuilder.Add(tn, string(t[model.LabelName(tn)]))
					gi++
					ti++
				}
			}
			for ; gi < len(groupSlice); gi++ {
				targetBuilder.Add(groupSlice[gi].Name, groupSlice[gi].Value)
			}
			for ; ti < len(targetLabelNames); ti++ {
				targetBuilder.Add(targetLabelNames[ti], string(t[model.LabelName(targetLabelNames[ti])]))
			}

			item := NewItem(jobName, string(t[model.AddressLabel]), targetBuilder.Labels(), "")
			intoTargets[index] = item
			index++
		}
	}
	m.targetsDiscovered.Record(context.Background(), count, metric.WithAttributes(attribute.String("job.name", jobName)))
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
