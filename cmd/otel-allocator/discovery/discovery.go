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

package discovery

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var (
	targetsDiscovered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets",
		Help: "Number of targets discovered.",
	}, []string{"job_name"})
)

type Manager struct {
	log        logr.Logger
	manager    *discovery.Manager
	logger     log.Logger
	close      chan struct{}
	configsMap map[allocatorWatcher.EventSource]*config.Config
	hook       discoveryHook
}

type discoveryHook interface {
	SetConfig(map[string][]*relabel.Config)
}

func NewManager(log logr.Logger, ctx context.Context, logger log.Logger, hook discoveryHook, options ...func(*discovery.Manager)) *Manager {
	manager := discovery.NewManager(ctx, logger, options...)

	go func() {
		if err := manager.Run(); err != nil {
			log.Error(err, "Discovery manager failed")
		}
	}()
	return &Manager{
		log:        log,
		manager:    manager,
		logger:     logger,
		close:      make(chan struct{}),
		configsMap: make(map[allocatorWatcher.EventSource]*config.Config),
		hook:       hook,
	}
}

func (m *Manager) GetScrapeConfigs() map[string]*config.ScrapeConfig {
	jobToScrapeConfig := map[string]*config.ScrapeConfig{}
	for _, c := range m.configsMap {
		for _, scrapeConfig := range c.ScrapeConfigs {
			jobToScrapeConfig[scrapeConfig.JobName] = scrapeConfig
		}
	}
	return jobToScrapeConfig
}

func (m *Manager) ApplyConfig(source allocatorWatcher.EventSource, cfg *config.Config) error {
	m.configsMap[source] = cfg

	discoveryCfg := make(map[string]discovery.Configs)
	relabelCfg := make(map[string][]*relabel.Config)

	for _, value := range m.configsMap {
		for _, scrapeConfig := range value.ScrapeConfigs {
			discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
			relabelCfg[scrapeConfig.JobName] = scrapeConfig.RelabelConfigs
		}
	}

	if m.hook != nil {
		m.hook.SetConfig(relabelCfg)
	}
	return m.manager.ApplyConfig(discoveryCfg)
}

func (m *Manager) Watch(fn func(targets map[string]*target.Item)) {
	log := m.log.WithValues("component", "opentelemetry-targetallocator")

	go func() {
		for {
			select {
			case <-m.close:
				log.Info("Service Discovery watch event stopped: discovery manager closed")
				return
			case tsets := <-m.manager.SyncCh():
				targets := map[string]*target.Item{}

				for jobName, tgs := range tsets {
					var count float64 = 0
					for _, tg := range tgs {
						for _, t := range tg.Targets {
							count++
							item := &target.Item{
								JobName:   jobName,
								TargetURL: string(t[model.AddressLabel]),
								Label:     t.Merge(tg.Labels),
							}
							targets[item.Hash()] = item
						}
					}
					targetsDiscovered.WithLabelValues(jobName).Set(count)
				}
				fn(targets)
			}
		}
	}()
}

func (m *Manager) Close() {
	close(m.close)
}
