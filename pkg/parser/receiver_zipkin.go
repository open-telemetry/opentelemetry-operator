package parser

const parserNameZipkin = "__zipkin"

// NewZipkinReceiverParser builds a new parser for Zipkin receivers
func NewZipkinReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 9411,
		parserName:  parserNameZipkin,
	}
}

func init() {
	Register("zipkin", NewZipkinReceiverParser)
}
