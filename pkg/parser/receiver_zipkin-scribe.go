package parser

const parserNameZipkinScribe = "__zipkinscribe"

// NewZipkinScribeReceiverParser builds a new parser for ZipkinScribe receivers
func NewZipkinScribeReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 9410,
		parserName:  parserNameZipkinScribe,
	}
}

func init() {
	Register("zipkin-scribe", NewZipkinScribeReceiverParser)
}
