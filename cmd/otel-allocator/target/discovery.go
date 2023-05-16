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
	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/relabel"

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
	configsMap           map[allocatorWatcher.EventSource]*config.Config
	hook                 discoveryHook
	scrapeConfigsHash    uint64
	scrapeConfigsUpdater scrapeConfigsUpdater
}

type discoveryHook interface {
	SetConfig(map[string][]*relabel.Config)
}

type scrapeConfigsUpdater interface {
	UpdateScrapeConfigResponse(map[string]*config.ScrapeConfig) error
}

func NewDiscoverer(log logr.Logger, manager *discovery.Manager, hook discoveryHook, scrapeConfigsUpdater scrapeConfigsUpdater) *Discoverer {
	return &Discoverer{
		log:                  log,
		manager:              manager,
		close:                make(chan struct{}),
		configsMap:           make(map[allocatorWatcher.EventSource]*config.Config),
		hook:                 hook,
		scrapeConfigsUpdater: scrapeConfigsUpdater,
	}
}

func (m *Discoverer) ApplyConfig(source allocatorWatcher.EventSource, cfg *config.Config) error {
	if cfg == nil {
		m.log.Info("Service Discovery got empty Prometheus config", "source", source.String())
		return nil
	}
	m.configsMap[source] = cfg
	jobToScrapeConfig := make(map[string]*config.ScrapeConfig)

	discoveryCfg := make(map[string]discovery.Configs)
	relabelCfg := make(map[string][]*relabel.Config)

	for _, value := range m.configsMap {
		for _, scrapeConfig := range value.ScrapeConfigs {
			jobToScrapeConfig[scrapeConfig.JobName] = scrapeConfig
			discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
			relabelCfg[scrapeConfig.JobName] = scrapeConfig.RelabelConfigs
		}
	}

	hash, err := hashstructure.Hash(jobToScrapeConfig, nil)
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
	for {
		select {
		case <-m.close:
			m.log.Info("Service Discovery watch event stopped: discovery manager closed")
			return nil
		case tsets := <-m.manager.SyncCh():
			targets := map[string]*Item{}

			for jobName, tgs := range tsets {
				var count float64 = 0
				for _, tg := range tgs {
					for _, t := range tg.Targets {
						count++
						item := NewItem(jobName, string(t[model.AddressLabel]), t.Merge(tg.Labels), "")
						targets[item.Hash()] = item
					}
				}
				targetsDiscovered.WithLabelValues(jobName).Set(count)
			}
			fn(targets)
		}
	}
}

func (m *Discoverer) Close() {
	close(m.close)
}
