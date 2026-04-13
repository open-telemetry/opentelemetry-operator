// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/healthcheck"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

func NewManagerOpts(log logr.Logger, cfg *config.Config, c client.Client, restCfg *rest.Config) bridgemanager.ManagerOpts {
	runtimes := make([]bridgemanager.Runtime, 0, len(cfg.Standalone.Agents))
	standaloneClient := NewClient(log.WithName("client"), c, restCfg, func() {
		for _, runtime := range runtimes {
			if err := runtime.Client.UpdateEffectiveConfig(context.Background()); err != nil {
				log.Error(err, "failed to update effective config", "agent", runtime.Name)
			}
		}
	})
	for _, configuredAgent := range cfg.Standalone.Agents {
		agentCfg := cfg.ForStandaloneAgent(configuredAgent)
		opampClient := agentCfg.CreateClient()
		applier := standaloneClient.ScopedApplier(configuredAgent)
		opampAgent := opampagent.NewAgent(log.WithName(configuredAgent.WorkloadRef.Name), applier, agentCfg, opampClient, proxy.NoopServer{})
		standaloneClient.RegisterHealthUpdater(configuredAgent, opampAgent.UpdateHealth)
		runtimes = append(runtimes, bridgemanager.Runtime{
			Name:       configuredAgent.WorkloadRef.Name,
			Client:     opampClient,
			OpAMPAgent: opampAgent,
		})
	}
	return bridgemanager.ManagerOpts{
		Log:                    log,
		Runtimes:               runtimes,
		HealthServer:           healthcheck.NewServer(log.WithName("healthcheck"), cfg.HealthListenAddr),
		KubernetesClient:       standaloneClient,
		PermissionReviewClient: c,
		ListRequiredPermissions: func() ([]bridgemanager.Permission, error) {
			return ListRequiredPermissions(cfg.Standalone.Agents, cfg.RemoteConfigEnabled())
		},
	}
}
