// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/healthcheck"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operatorbridge"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/standalone"
)

func main() {
	l := config.GetLogger()

	cfg, configLoadErr := config.Load(l, os.Args)
	if configLoadErr != nil {
		l.Error(configLoadErr, "Unable to load configuration")
		os.Exit(1)
	}
	l.Info("Starting the Remote Configuration service", "mode", cfg.Mode)

	kubeClient, kubeErr := cfg.GetKubernetesClient()
	if kubeErr != nil {
		l.Error(kubeErr, "Couldn't create kubernetes client")
		os.Exit(1)
	}

	// signalCtx is cancelled on interrupt, which stops the informer goroutine.
	signalCtx, cancelSignal := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelSignal()

	options := commonManagerOptions(l, cfg, kubeClient)
	if cfg.IsStandaloneMode() {
		options = append(options, standaloneManagerOptions(l, cfg, kubeClient)...)
	} else {
		options = append(options, operatorManagerOptions(l, cfg, kubeClient)...)
	}
	manager, err := bridgemanager.New(options...)
	if err != nil {
		l.Error(err, "Cannot configure OpAMP bridge")
		os.Exit(1)
	}
	if err := manager.Start(signalCtx); err != nil {
		l.Error(err, "Cannot start OpAMP bridge")
		os.Exit(1)
	}
	<-signalCtx.Done()
	manager.Shutdown()
}

func commonManagerOptions(log logr.Logger, cfg *config.Config, c client.Client) []bridgemanager.Option {
	return []bridgemanager.Option{
		bridgemanager.WithLogger(log),
		bridgemanager.WithHealthServer(healthcheck.NewServer(log.WithName("healthcheck"), cfg.HealthListenAddr)),
		bridgemanager.WithPermissionReviewClient(c),
	}
}

func standaloneManagerOptions(log logr.Logger, cfg *config.Config, c client.Client) []bridgemanager.Option {
	runtimes := make([]bridgemanager.Runtime, 0, len(cfg.Standalone.Agents))
	standaloneClient := standalone.NewClient(log.WithName("client"), c, cfg.GetRestConfig(), func() {
		for _, runtime := range runtimes {
			if err := runtime.Client.UpdateEffectiveConfig(context.Background()); err != nil {
				log.Error(err, "failed to update effective config", "agent", runtime.Name)
			}
		}
	})
	for _, configuredAgent := range cfg.Standalone.Agents {
		agentCfg := config.NewStandaloneAgentConfig(cfg, configuredAgent)
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
	return []bridgemanager.Option{
		bridgemanager.WithRuntimes(runtimes),
		bridgemanager.WithKubernetesClient(standaloneClient),
		bridgemanager.WithRequiredPermissions(func() ([]bridgemanager.Permission, error) {
			return standalone.ListRequiredPermissions(cfg.Standalone.Agents, cfg.RemoteConfigEnabled())
		}),
	}
}

func operatorManagerOptions(log logr.Logger, cfg *config.Config, c client.Client) []bridgemanager.Option {
	opampClient := cfg.CreateClient()
	applier := operator.NewClient(cfg.Name, log.WithName("operator-client"), c, cfg.GetComponentsAllowed())
	opampProxy := proxy.NewOpAMPProxy(log.WithName("server"), cfg.ListenAddr)
	opampAgent := opampagent.NewAgent(log.WithName("agent"), applier, cfg, opampClient, opampProxy)
	return []bridgemanager.Option{
		bridgemanager.WithOpAMPProxy(opampProxy),
		bridgemanager.WithRequiredPermissions(operatorbridge.ListRequiredPermissions),
		bridgemanager.WithRuntimes([]bridgemanager.Runtime{
			{
				Name:       cfg.Name,
				Client:     opampClient,
				OpAMPAgent: opampAgent,
			},
		}),
	}
}
