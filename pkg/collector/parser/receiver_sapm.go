package parser

import "github.com/go-logr/logr"

const parserNameSAPM = "__sapm"

// NewSAPMReceiverParser builds a new parser for SAPM receivers, from the contrib repository
func NewSAPMReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 7276,
		parserName:  parserNameSAPM,
	}
}

func init() {
	Register("sapm", NewSAPMReceiverParser)
}
