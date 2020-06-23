package parser

const parserNameSignalFx = "__signalfx"

// NewSignalFxReceiverParser builds a new parser for SignalFx receivers, from the contrib repository
func NewSignalFxReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 7276,
		parserName:  parserNameSignalFx,
	}
}

func init() {
	Register("signalfx", NewSignalFxReceiverParser)
}
