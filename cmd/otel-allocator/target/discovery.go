// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var (
	targetsDiscovered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets",
		Help: "Number of targets discovered.",
	}, []string{"job_name"})

	processTargetsDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "opentelemetry_allocator_process_targets_duration_seconds",
		Help:    "Duration of processing targets.",
		Buckets: []float64{1, 5, 10, 30, 60, 120},
	})

	processTargetGroupsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "opentelemetry_allocator_process_target_groups_duration_seconds",
		Help:    "Duration of processing target groups.",
		Buckets: []float64{1, 5, 10, 30, 60, 120},
	}, []string{"job_name"})
)

type Discoverer struct {
	log                    logr.Logger
	manager                *discovery.Manager
	close                  chan struct{}
	mtxScrape              sync.Mutex // Guards the fields below.
	configsMap             map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig
	hook                   discoveryHook
	scrapeConfigsHash      hash.Hash
	scrapeConfigsUpdater   scrapeConfigsUpdater
	targetSets             map[string][]*targetgroup.Group
	triggerReload          chan struct{}
	processTargetsCallBack func(targets map[string]*Item)
	mtxTargets             sync.Mutex
}

type discoveryHook interface {
	SetConfig(map[string][]*relabel.Config)
}

type scrapeConfigsUpdater interface {
	UpdateScrapeConfigResponse(map[string]*promconfig.ScrapeConfig) error
}

func NewDiscoverer(log logr.Logger, manager *discovery.Manager, hook discoveryHook, scrapeConfigsUpdater scrapeConfigsUpdater, setTargets func(targets map[string]*Item)) *Discoverer {
	return &Discoverer{
		log:                    log,
		manager:                manager,
		close:                  make(chan struct{}),
		triggerReload:          make(chan struct{}, 1),
		configsMap:             make(map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig),
		hook:                   hook,
		scrapeConfigsHash:      nil, // we want the first update to succeed even if the config is empty
		scrapeConfigsUpdater:   scrapeConfigsUpdater,
		processTargetsCallBack: setTargets,
	}
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
	// Otherwise skip updating scrape configs.
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
	targets := map[string]*Item{}
	timer := prometheus.NewTimer(processTargetsDuration)
	defer timer.ObserveDuration()

	for jobName, groups := range m.targetSets {
		wg.Add(1)
		// Run the sync in parallel as these take a while and at high load can't catch up.
		go func(jobName string, groups []*targetgroup.Group) {
			processedTargets := m.processTargetGroups(jobName, groups)
			m.mtxTargets.Lock()
			for k, v := range processedTargets {
				targets[k] = v
			}
			m.mtxTargets.Unlock()
			wg.Done()
		}(jobName, groups)
	}
	m.mtxScrape.Unlock()
	wg.Wait()
	m.processTargetsCallBack(targets)
}

// processTargetGroups processes the target groups and returns a map of targets.
func (m *Discoverer) processTargetGroups(jobName string, groups []*targetgroup.Group) map[string]*Item {
	builder := labels.NewBuilder(labels.Labels{})
	timer := prometheus.NewTimer(processTargetGroupsDuration.WithLabelValues(jobName))
	targets := map[string]*Item{}
	defer timer.ObserveDuration()
	var count float64 = 0
	for _, tg := range groups {
		builder.Reset(labels.EmptyLabels())
		for ln, lv := range tg.Labels {
			builder.Set(string(ln), string(lv))
		}
		groupLabels := builder.Labels()
		for _, t := range tg.Targets {
			count++
			builder.Reset(groupLabels)
			for ln, lv := range t {
				builder.Set(string(ln), string(lv))
			}
			item := NewItem(jobName, string(t[model.AddressLabel]), builder.Labels(), "")
			targets[item.Hash()] = item
		}
	}
	targetsDiscovered.WithLabelValues(jobName).Set(count)
	return targets
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
	yamlEncoder := yaml.NewEncoder(hash)
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
