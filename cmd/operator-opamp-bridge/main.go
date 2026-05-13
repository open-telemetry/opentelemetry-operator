// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
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

	if cfg.IsStandaloneMode() {
		standaloneManager := standalone.NewManager(l.WithName("standalone"), cfg, kubeClient, cfg.GetRestConfig())
		if err := standaloneManager.Start(signalCtx); err != nil {
			l.Error(err, "Cannot start standalone agents")
			os.Exit(1)
		}
		<-signalCtx.Done()
		standaloneManager.Shutdown()
		return
	}

	opampClient := cfg.CreateClient()
	applier := operator.NewClient(cfg.Name, l.WithName("operator-client"), kubeClient, cfg.GetComponentsAllowed())
	opampProxy := proxy.NewOpAMPProxy(l.WithName("server"), cfg.ListenAddr)
	opampAgent := agent.NewAgent(l.WithName("agent"), applier, cfg, opampClient, opampProxy)

	if err := opampAgent.Start(); err != nil {
		l.Error(err, "Cannot start OpAMP client")
		os.Exit(1)
	}
	if err := opampProxy.Start(); err != nil {
		l.Error(err, "failed to start OpAMP Server")
		os.Exit(1)
	}

	<-signalCtx.Done()
	opampAgent.Shutdown()
	proxyStopErr := opampProxy.Stop(context.Background())
	if proxyStopErr != nil {
		l.Error(proxyStopErr, "failed to shutdown proxy server")
	}
}
