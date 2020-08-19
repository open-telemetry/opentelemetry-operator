package parser

import "github.com/go-logr/logr"

const parserNameSignalFx = "__signalfx"

// NewSignalFxReceiverParser builds a new parser for SignalFx receivers, from the contrib repository
func NewSignalFxReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 9943,
		parserName:  parserNameSignalFx,
	}
}

func init() {
	Register("signalfx", NewSignalFxReceiverParser)
}
