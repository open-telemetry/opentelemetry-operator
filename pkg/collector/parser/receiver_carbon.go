package parser

import "github.com/go-logr/logr"

const parserNameCarbon = "__carbon"

// NewCarbonReceiverParser builds a new parser for Carbon receivers, from the contrib repository
func NewCarbonReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 2003,
		parserName:  parserNameCarbon,
	}
}

func init() {
	Register("carbon", NewCarbonReceiverParser)
}
