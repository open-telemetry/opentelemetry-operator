package parser

const parserNameCollectd = "__collectd"

// NewCollectdReceiverParser builds a new parser for Collectd receivers, from the contrib repository
func NewCollectdReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 8081,
		parserName:  parserNameCollectd,
	}
}

func init() {
	Register("collectd", NewCollectdReceiverParser)
}
