package parser

const parserNameOTLP = "__otlp"

// NewOTLPReceiverParser builds a new parser for OTLP receivers
func NewOTLPReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 55680,
		parserName:  parserNameOTLP,
	}
}

func init() {
	Register("otlp", NewOTLPReceiverParser)
}
