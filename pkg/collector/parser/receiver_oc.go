package parser

import "github.com/go-logr/logr"

const parserNameOpenCensus = "__opencensus"

// NewOpenCensusReceiverParser builds a new parser for OpenCensus receivers
func NewOpenCensusReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 55678,
		parserName:  parserNameOpenCensus,
	}
}

func init() {
	Register("opencensus", NewOpenCensusReceiverParser)
}
