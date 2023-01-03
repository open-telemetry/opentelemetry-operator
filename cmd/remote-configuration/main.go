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

	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/logger"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/operator"

	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/config"
)

func main() {
	cliConf, err := config.ParseCLI()
	if err != nil {
		os.Exit(1)
	}
	cfg, configLoadErr := config.Load(*cliConf.ConfigFilePath)
	if configLoadErr != nil {
		cliConf.RootLogger.Error(configLoadErr, "Unable to load configuration")
		return
	}
	cliConf.RootLogger.Info("Starting the Remote Configuration service")
	agentLogf := cliConf.RootLogger.WithName("agent")
	agentLogger := logger.NewLogger(&agentLogf)

	kubeClient, kubeErr := cliConf.GetKubernetesClient()
	if kubeErr != nil {
		cliConf.RootLogger.Error(kubeErr, "Couldn't create kubernetes client")
		return
	}
	operatorClient := operator.NewClient(cliConf.RootLogger.WithName("operator-client"), kubeClient, cfg.GetComponentsAllowed())

	opampClient := cfg.CreateClient(agentLogger)
	opampAgent := agent.NewAgent(agentLogger, operatorClient, cfg, opampClient)

	if err := opampAgent.Start(); err != nil {
		cliConf.RootLogger.Error(err, "Cannot start OpAMP client")
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
}
