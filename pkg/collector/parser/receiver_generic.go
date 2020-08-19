package parser

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

const parserNameGeneric = "__generic"

var _ ReceiverParser = &GenericReceiver{}

// GenericReceiver is a special parser for generic receivers. It doesn't self-register and should be created/used directly
type GenericReceiver struct {
	logger      logr.Logger
	name        string
	config      map[interface{}]interface{}
	defaultPort int32
	parserName  string
}

// NewGenericReceiverParser builds a new parser for generic receivers
func NewGenericReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:     logger,
		name:       name,
		config:     config,
		parserName: parserNameGeneric,
	}
}

// Ports returns all the service ports for all protocols in this parser
func (g *GenericReceiver) Ports() ([]corev1.ServicePort, error) {
	port := singlePortFromConfigEndpoint(g.logger, g.name, g.config)
	if port != nil {
		return []corev1.ServicePort{*port}, nil
	}

	if g.defaultPort > 0 {
		return []corev1.ServicePort{{
			Port: g.defaultPort,
			Name: portName(g.name, g.defaultPort),
		}}, nil
	}

	return []corev1.ServicePort{}, nil
}

// ParserName returns the name of this parser
func (g *GenericReceiver) ParserName() string {
	return g.parserName
}
