package discovery

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
)

type Manager struct {
	log     logr.Logger
	manager *discovery.Manager
	logger  log.Logger
	close   chan struct{}
}

func NewManager(log logr.Logger, ctx context.Context, logger log.Logger, options ...func(*discovery.Manager)) *Manager {
	manager := discovery.NewManager(ctx, logger, options...)

	go func() {
		if err := manager.Run(); err != nil {
			logger.Log("Discovery manager failed", err)
		}
	}()
	return &Manager{
		log:     log,
		manager: manager,
		logger:  logger,
		close:   make(chan struct{}),
	}
}

func (m *Manager) ApplyConfig(cfg config.Config) error {
	discoveryCfg := make(map[string]discovery.Configs)

	for _, scrapeConfig := range cfg.Config.ScrapeConfigs {
		discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
	}
	return m.manager.ApplyConfig(discoveryCfg)
}

func (m *Manager) Watch(fn func(targets []allocation.TargetItem)) {
	log := m.log.WithValues("opentelemetry-targetallocator")

	go func() {
		for {
			select {
			case <-m.close:
				log.Info("Service Discovery watch event stopped: discovery manager closed")
				return
			case tsets := <-m.manager.SyncCh():
				targets := []allocation.TargetItem{}

				for jobName, tgs := range tsets {
					for _, tg := range tgs {
						for _, t := range tg.Targets {
							targets = append(targets, allocation.TargetItem{
								JobName:   jobName,
								TargetURL: string(t[model.AddressLabel]),
								Label:     tg.Labels,
							})
						}
					}
				}
				fn(targets)
			}
		}
	}()
}

func (m *Manager) Close() {
	close(m.close)
}
