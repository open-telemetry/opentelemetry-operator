package parser

import "github.com/go-logr/logr"

const parserNameZipkin = "__zipkin"

// NewZipkinReceiverParser builds a new parser for Zipkin receivers
func NewZipkinReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 9411,
		parserName:  parserNameZipkin,
	}
}

func init() {
	Register("zipkin", NewZipkinReceiverParser)
}
