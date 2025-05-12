package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/open-telemetry/opentelemetry-operator/tests/test-e2e-apps/bridge-server/data"
	"github.com/open-telemetry/opentelemetry-operator/tests/test-e2e-apps/bridge-server/opampsrv"
)

var logger = log.New(log.Default().Writer(), "[MAIN] ", log.Default().Flags()|log.Lmsgprefix|log.Lmicroseconds)

func main() {

	logger.Println("OpAMP Server starting...")
	agents := data.NewAgents()
	opampSrv := opampsrv.NewServer(agents)
	opampSrv.Start()

	logger.Println("OpAMP Server running...")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	logger.Println("OpAMP Server shutting down...")
	opampSrv.Stop()
}
