package parser

const parserNameWavefront = "__wavefront"

// NewWavefrontReceiverParser builds a new parser for Wavefront receivers, from the contrib repository
func NewWavefrontReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 2003,
		parserName:  parserNameWavefront,
	}
}

func init() {
	Register("wavefront", NewWavefrontReceiverParser)
}
