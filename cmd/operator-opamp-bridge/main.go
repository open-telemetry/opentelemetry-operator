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

package main

import (
	"os"
	"os/signal"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/logger"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"
)

func main() {
	l := config.GetLogger()
	cliConf, err := config.ParseCLI(l.WithName("cli-config"))
	if err != nil {
		l.Error(err, "unable to load ")
		os.Exit(1)
	}
	cfg, configLoadErr := config.Load(*cliConf.ConfigFilePath)
	if configLoadErr != nil {
		l.Error(configLoadErr, "Unable to load configuration")
		return
	}
	l.Info("Starting the Remote Configuration service")
	agentLogf := l.WithName("agent")
	agentLogger := logger.NewLogger(&agentLogf)

	kubeClient, kubeErr := cliConf.GetKubernetesClient()
	if kubeErr != nil {
		l.Error(kubeErr, "Couldn't create kubernetes client")
		os.Exit(1)
	}
	operatorClient := operator.NewClient(l.WithName("operator-client"), kubeClient, cfg.GetComponentsAllowed())

	opampClient := cfg.CreateClient(agentLogger)
	opampAgent := agent.NewAgent(agentLogger, operatorClient, cfg, opampClient)

	if err := opampAgent.Start(); err != nil {
		l.Error(err, "Cannot start OpAMP client")
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
}
