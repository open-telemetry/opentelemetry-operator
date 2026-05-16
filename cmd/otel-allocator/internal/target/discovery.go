// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"context"
	"hash"
	"hash/fnv"
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
	jobToScrapeConfig           map[string]*promconfig.ScrapeConfig
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
		err = m.scrapeConfigsUpdater.UpdateScrapeConfigResponse(jobToScrapeConfig)
		if err != nil {
			return err
		}

		m.scrapeConfigsHash = hash
	}

	if m.hook != nil {
		m.hook.SetConfig(relabelCfg)
	}

	err = m.manager.ApplyConfig(discoveryCfg)
	if err != nil {
		return err
	}

	// Store scrape configs only after all operations succeed to avoid
	// inconsistent state on partial failure. Guard with mtxScrape since
	// Reload() reads this field concurrently under the same lock.
	m.mtxScrape.Lock()
	m.jobToScrapeConfig = jobToScrapeConfig
	m.mtxScrape.Unlock()

	return nil
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

	jobToScrapeConfig := m.jobToScrapeConfig
	targetsAssigned := 0
	for jobName, groups := range m.targetSets {
		wg.Add(1)
		scrapeConfig := jobToScrapeConfig[jobName]
		// Run the sync in parallel as these take a while and at high load can't catch up.
		go func(jobName string, groups []*targetgroup.Group, intoTargets []*Item, cfg *promconfig.ScrapeConfig) {
			m.processTargetGroups(jobName, groups, intoTargets, cfg)
			wg.Done()
		}(jobName, groups, targets[targetsAssigned:], scrapeConfig)
		for _, group := range groups {
			targetsAssigned += len(group.Targets)
		}
	}
	m.mtxScrape.Unlock()
	wg.Wait()
	m.processTargetsCallBack(targets)
}

// processTargetGroups processes the target groups and populates labels the same way
// Prometheus does (via scrape.PopulateDiscoveredLabels) before creating target items.
// This ensures the label set includes scrape config defaults (job, metrics_path,
// scheme, scrape_interval, scrape_timeout) so that target hashes are consistent
// with what Prometheus computes.
func (m *Discoverer) processTargetGroups(jobName string, groups []*targetgroup.Group, intoTargets []*Item, cfg *promconfig.ScrapeConfig) {
	lb := labels.NewBuilder(labels.EmptyLabels())
	groupBuilder := labels.NewScratchBuilder(labelBuilderPreallocSize)
	defaults := newDiscoveredLabelDefaults(cfg)

	begin := time.Now()
	defer func() {
		m.processTargetGroupsDuration.Record(context.Background(), time.Since(begin).Seconds(), metric.WithAttributes(attribute.String("job.name", jobName)))
	}()
	var count float64
	index := 0
	for _, tg := range groups {
		groupLabels := populateGroupLabels(&groupBuilder, tg.Labels)
		for _, t := range tg.Targets {
			count++
			lset := populateDiscoveredLabels(lb, defaults, t, groupLabels)
			item := NewItem(jobName, string(t[model.AddressLabel]), lset, "")
			intoTargets[index] = item
			index++
		}
	}
	m.targetsDiscovered.Record(context.Background(), count, metric.WithAttributes(attribute.String("job.name", jobName)))
}

func populateGroupLabels(groupBuilder *labels.ScratchBuilder, tgLabels model.LabelSet) labels.Labels {
	groupBuilder.Reset()
	for ln, lv := range tgLabels {
		if lv != "" {
			groupBuilder.Add(string(ln), string(lv))
		}
	}
	groupBuilder.Sort()
	return groupBuilder.Labels()
}

type discoveredLabelDefaults struct {
	jobName        string
	scrapeInterval string
	scrapeTimeout  string
	metricsPath    string
	scheme         string
	params         []labels.Label
}

func newDiscoveredLabelDefaults(cfg *promconfig.ScrapeConfig) discoveredLabelDefaults {
	defaults := discoveredLabelDefaults{
		jobName:        cfg.JobName,
		scrapeInterval: cfg.ScrapeInterval.String(),
		scrapeTimeout:  cfg.ScrapeTimeout.String(),
		metricsPath:    cfg.MetricsPath,
		scheme:         cfg.Scheme,
	}

	if len(cfg.Params) > 0 {
		defaults.params = make([]labels.Label, 0, len(cfg.Params))
		for k, v := range cfg.Params {
			if len(v) > 0 {
				defaults.params = append(defaults.params, labels.Label{
					Name:  model.ParamLabelPrefix + k,
					Value: v[0],
				})
			}
		}
	}

	return defaults
}

// populateDiscoveredLabels matches scrape.PopulateDiscoveredLabels. Keep this
// local hot-path helper in sync with Prometheus and covered by the comparison
// test, because calling the Prometheus helper directly repeats group-label and
// scrape-default work for every target.
func populateDiscoveredLabels(lb *labels.Builder, defaults discoveredLabelDefaults, tLabels model.LabelSet, groupLabels labels.Labels) labels.Labels {
	lb.Reset(groupLabels)
	for ln, lv := range tLabels {
		lb.Set(string(ln), string(lv))
	}

	setDefaultLabel(lb, model.JobLabel, defaults.jobName)
	setDefaultLabel(lb, model.ScrapeIntervalLabel, defaults.scrapeInterval)
	setDefaultLabel(lb, model.ScrapeTimeoutLabel, defaults.scrapeTimeout)
	setDefaultLabel(lb, model.MetricsPathLabel, defaults.metricsPath)
	setDefaultLabel(lb, model.SchemeLabel, defaults.scheme)

	for _, param := range defaults.params {
		setDefaultLabel(lb, param.Name, param.Value)
	}

	return lb.Labels()
}

func setDefaultLabel(lb *labels.Builder, name, value string) {
	if lb.Get(name) == "" {
		lb.Set(name, value)
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
