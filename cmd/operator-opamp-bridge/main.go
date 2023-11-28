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

	"github.com/spf13/pflag"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"
)

func main() {
	l := config.GetLogger()

	flagSet := config.GetFlagSet(pflag.ExitOnError)
	err := flagSet.Parse(os.Args)
	if err != nil {
		l.Error(err, "Unable to load flags")
		os.Exit(1)
	}
	cfg, configLoadErr := config.Load(l, flagSet)
	if configLoadErr != nil {
		l.Error(configLoadErr, "Unable to load configuration")
		os.Exit(1)
	}
	l.Info("Starting the Remote Configuration service")

	kubeClient, kubeErr := cfg.GetKubernetesClient()
	if kubeErr != nil {
		l.Error(kubeErr, "Couldn't create kubernetes client")
		os.Exit(1)
	}
	operatorClient := operator.NewClient(cfg.Name, l.WithName("operator-client"), kubeClient, cfg.GetComponentsAllowed())

	opampClient := cfg.CreateClient()
	opampAgent := agent.NewAgent(l.WithName("agent"), operatorClient, cfg, opampClient)

	if err := opampAgent.Start(); err != nil {
		l.Error(err, "Cannot start OpAMP client")
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
}
