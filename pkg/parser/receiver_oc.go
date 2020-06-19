package parser

const parserNameOpenCensus = "__opencensus"

// NewOpenCensusReceiverParser builds a new parser for Zipkin receivers
func NewOpenCensusReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 55678,
		parserName:  parserNameOpenCensus,
	}
}

func init() {
	Register("opencensus", NewOpenCensusReceiverParser)
}
