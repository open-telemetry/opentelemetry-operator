package parser

import "github.com/go-logr/logr"

const parserNameWavefront = "__wavefront"

// NewWavefrontReceiverParser builds a new parser for Wavefront receivers, from the contrib repository
func NewWavefrontReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 2003,
		parserName:  parserNameWavefront,
	}
}

func init() {
	Register("wavefront", NewWavefrontReceiverParser)
}
