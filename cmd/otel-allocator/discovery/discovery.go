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

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
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
}

func NewManager(log logr.Logger, ctx context.Context, logger log.Logger, options ...func(*discovery.Manager)) *Manager {
	manager := discovery.NewManager(ctx, logger, options...)

	go func() {
		if err := manager.Run(); err != nil {
			logger.Log("Discovery manager failed", err)
		}
	}()
	return &Manager{
		log:        log,
		manager:    manager,
		logger:     logger,
		close:      make(chan struct{}),
		configsMap: make(map[allocatorWatcher.EventSource]*config.Config),
	}
}

func (m *Manager) ApplyConfig(source allocatorWatcher.EventSource, cfg *config.Config) error {
	m.configsMap[source] = cfg

	discoveryCfg := make(map[string]discovery.Configs)

	for _, value := range m.configsMap {
		for _, scrapeConfig := range value.ScrapeConfigs {
			discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
		}
	}
	return m.manager.ApplyConfig(discoveryCfg)
}

func (m *Manager) Watch(fn func(targets map[string]*allocation.TargetItem)) {
	log := m.log.WithValues("component", "opentelemetry-targetallocator")

	go func() {
		for {
			select {
			case <-m.close:
				log.Info("Service Discovery watch event stopped: discovery manager closed")
				return
			case tsets := <-m.manager.SyncCh():
				targets := map[string]*allocation.TargetItem{}

				for jobName, tgs := range tsets {
					var count float64 = 0
					for _, tg := range tgs {
						for _, t := range tg.Targets {
							count++
							item := &allocation.TargetItem{
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
