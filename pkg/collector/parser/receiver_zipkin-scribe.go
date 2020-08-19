package parser

import "github.com/go-logr/logr"

const parserNameZipkinScribe = "__zipkinscribe"

// NewZipkinScribeReceiverParser builds a new parser for ZipkinScribe receivers
func NewZipkinScribeReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 9410,
		parserName:  parserNameZipkinScribe,
	}
}

func init() {
	Register("zipkin-scribe", NewZipkinScribeReceiverParser)
}
