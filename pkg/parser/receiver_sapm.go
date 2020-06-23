package parser

const parserNameSAPM = "__sapm"

// NewSAPMReceiverParser builds a new parser for SAPM receivers, from the contrib repository
func NewSAPMReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &GenericReceiver{
		name:        name,
		config:      config,
		defaultPort: 7276,
		parserName:  parserNameSAPM,
	}
}

func init() {
	Register("sapm", NewSAPMReceiverParser)
}
