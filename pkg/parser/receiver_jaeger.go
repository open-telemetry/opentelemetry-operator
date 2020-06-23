package parser

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

var _ ReceiverParser = &JaegerReceiverParser{}

const (
	parserNameJaeger = "__jaeger"

	defaultGRPCPort          int32 = 14250
	defaultThriftHTTPPort    int32 = 14268
	defaultThriftCompactPort int32 = 6831
	defaultThriftBinaryPort  int32 = 6832
)

// JaegerReceiverParser parses the configuration for Jaeger-specific receivers
type JaegerReceiverParser struct {
	name   string
	config map[interface{}]interface{}
}

// NewJaegerReceiverParser builds a new parser for Jaeger receivers
func NewJaegerReceiverParser(name string, config map[interface{}]interface{}) ReceiverParser {
	return &JaegerReceiverParser{
		name:   name,
		config: config,
	}
}

// Ports returns all the service ports for all protocols in this parser
func (j *JaegerReceiverParser) Ports(ctx context.Context) ([]corev1.ServicePort, error) {
	ports := []corev1.ServicePort{}

	for _, protocol := range []struct {
		name        string
		defaultPort int32
	}{
		{
			name:        "grpc",
			defaultPort: defaultGRPCPort,
		},
		{
			name:        "thrift_http",
			defaultPort: defaultThriftHTTPPort,
		},
		{
			name:        "thrift_compact",
			defaultPort: defaultThriftCompactPort,
		},
		{
			name:        "thrift_binary",
			defaultPort: defaultThriftBinaryPort,
		},
	} {
		// do we have the protocol specified at all?
		if receiverProtocol, ok := j.config[protocol.name]; ok {
			// we have the specified protocol, we definitely need a service port
			nameWithProtocol := fmt.Sprintf("%s-%s", j.name, protocol.name)
			var protocolPort *corev1.ServicePort

			// do we have a configuration block for the protocol?
			settings, ok := receiverProtocol.(map[interface{}]interface{})
			if ok {
				protocolPort = singlePortFromConfigEndpoint(ctx, nameWithProtocol, settings)
			}

			// have we parsed a port based on the configuration block?
			// if not, we use the default port
			if protocolPort == nil {
				protocolPort = &corev1.ServicePort{
					Name: portName(nameWithProtocol, protocol.defaultPort),
					Port: protocol.defaultPort,
				}
			}

			// at this point, we *have* a port specified, add it to the list of ports
			ports = append(ports, *protocolPort)
		}
	}

	return ports, nil
}

// ParserName returns the name of this parser
func (j *JaegerReceiverParser) ParserName() string {
	return parserNameJaeger
}

func init() {
	Register("jaeger", NewJaegerReceiverParser)
}
