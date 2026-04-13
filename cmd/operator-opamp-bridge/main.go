// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operatorbridge"
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

	newManagerOpts := operatorbridge.NewManagerOpts
	if cfg.IsStandaloneMode() {
		newManagerOpts = standalone.NewManagerOpts
	}
	opts := newManagerOpts(l, cfg, kubeClient, cfg.GetRestConfig())
	manager := bridgemanager.New(opts)
	if err := manager.Start(signalCtx); err != nil {
		l.Error(err, "Cannot start OpAMP bridge")
		os.Exit(1)
	}
	<-signalCtx.Done()
	manager.Shutdown()
}
