// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	opampclient "github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/protobufs"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
)

type Manager struct {
	log     logr.Logger
	cfg     *config.Config
	client  *Client
	runtime []agentRuntime
}

type agentRuntime struct {
	name       string
	opampAgent *opampagent.Agent
	client     opampclient.OpAMPClient
}

func NewManager(log logr.Logger, cfg *config.Config, c client.Client, restCfg *rest.Config) *Manager {
	manager := &Manager{
		log: log,
		cfg: cfg,
	}
	manager.client = NewClient(cfg.Name, log.WithName("client"), c, restCfg, manager.updateEffectiveConfig)
	return manager
}

func (m *Manager) Start(ctx context.Context) error {
	if err := m.client.Start(ctx); err != nil {
		return err
	}
	for _, configuredAgent := range m.cfg.Standalone.Agents {
		agentCfg := m.cfg.ForStandaloneAgent(configuredAgent)
		opampClient := agentCfg.CreateClient()
		runtime := agentRuntime{
			name:       configuredAgent.Name,
			client:     opampClient,
			opampAgent: opampagent.NewAgent(m.log.WithName(configuredAgent.Name), m.client.scopedApplier(configuredAgent), agentCfg, opampClient, noopProxy{}),
		}
		if err := runtime.opampAgent.Start(); err != nil {
			m.Shutdown()
			return err
		}
		m.runtime = append(m.runtime, runtime)
	}
	return nil
}

func (m *Manager) Shutdown() {
	for _, runtime := range m.runtime {
		runtime.opampAgent.Shutdown()
	}
}

func (m *Manager) updateEffectiveConfig() {
	for _, runtime := range m.runtime {
		if err := runtime.client.UpdateEffectiveConfig(context.Background()); err != nil {
			m.log.Error(err, "failed to update effective config after ConfigMap change", "agent", runtime.name)
		}
	}
}

type noopProxy struct{}

func (noopProxy) GetAgentsByHostname() map[string]uuid.UUID {
	return map[string]uuid.UUID{}
}

func (noopProxy) GetConfigurations() map[uuid.UUID]*protobufs.EffectiveConfig {
	return map[uuid.UUID]*protobufs.EffectiveConfig{}
}

func (noopProxy) GetHealth() map[uuid.UUID]*protobufs.ComponentHealth {
	return map[uuid.UUID]*protobufs.ComponentHealth{}
}

func (noopProxy) HasUpdates() <-chan struct{} {
	return make(chan struct{})
}
