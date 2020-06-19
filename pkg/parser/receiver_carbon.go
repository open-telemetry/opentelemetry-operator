package parser

const parserNameCarbon = "__carbon"

// NewCarbonReceiverParser builds a new parser for Carbon receivers, from the contrib repository
func NewCarbonReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 2003,
		parserName:  parserNameCarbon,
	}
}

func init() {
	Register("carbon", NewCarbonReceiverParser)
}
