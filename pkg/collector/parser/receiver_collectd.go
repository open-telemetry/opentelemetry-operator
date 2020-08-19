package parser

import "github.com/go-logr/logr"

const parserNameCollectd = "__collectd"

// NewCollectdReceiverParser builds a new parser for Collectd receivers, from the contrib repository
func NewCollectdReceiverParser(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		logger:      logger,
		name:        name,
		config:      config,
		defaultPort: 8081,
		parserName:  parserNameCollectd,
	}
}

func init() {
	Register("collectd", NewCollectdReceiverParser)
}
