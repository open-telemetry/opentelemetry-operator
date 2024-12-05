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

package target

import (
	"hash"
	"hash/fnv"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v3"

	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var (
	targetsDiscovered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets",
		Help: "Number of targets discovered.",
	}, []string{"job_name"})
)

type Discoverer struct {
	log                  logr.Logger
	manager              *discovery.Manager
	close                chan struct{}
	configsMap           map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig
	hook                 discoveryHook
	scrapeConfigsHash    hash.Hash
	scrapeConfigsUpdater scrapeConfigsUpdater
}

type discoveryHook interface {
	SetConfig(map[string][]*relabel.Config)
}

type scrapeConfigsUpdater interface {
	UpdateScrapeConfigResponse(map[string]*promconfig.ScrapeConfig) error
}

func NewDiscoverer(log logr.Logger, manager *discovery.Manager, hook discoveryHook, scrapeConfigsUpdater scrapeConfigsUpdater) *Discoverer {
	return &Discoverer{
		log:                  log,
		manager:              manager,
		close:                make(chan struct{}),
		configsMap:           make(map[allocatorWatcher.EventSource][]*promconfig.ScrapeConfig),
		hook:                 hook,
		scrapeConfigsHash:    nil, // we want the first update to succeed even if the config is empty
		scrapeConfigsUpdater: scrapeConfigsUpdater,
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

func (m *Discoverer) Watch(fn func(targets map[string]*Item)) error {
	labelsBuilder := labels.NewBuilder(labels.EmptyLabels())
	for {
		select {
		case <-m.close:
			m.log.Info("Service Discovery watch event stopped: discovery manager closed")
			return nil
		case tsets := <-m.manager.SyncCh():
			m.ProcessTargets(labelsBuilder, tsets, fn)
		}
	}
}

func (m *Discoverer) ProcessTargets(builder *labels.Builder, tsets map[string][]*targetgroup.Group, fn func(targets map[string]*Item)) {
	targets := map[string]*Item{}

	for jobName, tgs := range tsets {
		var count float64 = 0
		for _, tg := range tgs {
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
	}
	fn(targets)
}

func (m *Discoverer) Close() {
	close(m.close)
}

// Calculate a hash for a scrape config map.
// This is done by marshaling to YAML because it's the most straightforward and doesn't run into problems with unexported fields.
func getScrapeConfigHash(jobToScrapeConfig map[string]*promconfig.ScrapeConfig) (hash.Hash64, error) {
	var err error
	hash := fnv.New64()
	yamlEncoder := yaml.NewEncoder(hash)
	for jobName, scrapeConfig := range jobToScrapeConfig {
		_, err = hash.Write([]byte(jobName))
		if err != nil {
			return nil, err
		}
		err = yamlEncoder.Encode(scrapeConfig)
		if err != nil {
			return nil, err
		}
	}
	yamlEncoder.Close()
	return hash, err
}
