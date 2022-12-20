package main

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/config"
	"os"
	"os/signal"
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

	opampAgent := agent.NewAgent(agent.NewLogger(&agentLogger), cfg, *cliConf.AgentType, *cliConf.AgentVersion)

	if err := opampAgent.Start(); err != nil {
		cliConf.RootLogger.Error(err, "Cannot start OpAMP client")
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
}
