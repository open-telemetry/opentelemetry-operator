package parser

import "github.com/go-logr/logr"

const parserNameOTLP = "__otlp"

// NewOTLPReceiverParser builds a new parser for OTLP receivers
func NewOTLPReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 55680,
		parserName:  parserNameOTLP,
	}
}

func init() {
	Register("otlp", NewOTLPReceiverParser)
}
