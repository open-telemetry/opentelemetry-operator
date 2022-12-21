package main

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/operator"
	"os"
	"os/signal"

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
	agentLogger := cliConf.RootLogger.WithName("agent")

	operatorClient, operatorClientErr := operator.NewClient(cliConf.RootLogger.WithName("operator-client"), cliConf.ClusterConfig)
	if operatorClientErr != nil {
		cliConf.RootLogger.Error(operatorClientErr, "Couldn't create operator client")
	}
	opampAgent := agent.NewAgent(agent.NewLogger(&agentLogger), operatorClient, cfg, *cliConf.AgentType, *cliConf.AgentVersion)

	if err := opampAgent.Start(); err != nil {
		cliConf.RootLogger.Error(err, "Cannot start OpAMP client")
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
}
