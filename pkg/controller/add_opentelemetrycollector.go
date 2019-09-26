package controller

import (
	"github.com/open-telemetry/opentelemetry-operator/pkg/controller/opentelemetrycollector"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, opentelemetrycollector.Add)
}
