package adapters

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	// ErrNoReceivers indicates that there are no receivers in the configuration
	ErrNoReceivers = errors.New("no receivers available as part of the configuration")

	// ErrReceiversNotAMap indicates that the receivers property isn't a map of values
	ErrReceiversNotAMap = errors.New("receivers property in the configuration doesn't contain valid receivers")

	// DNS_LABEL constraints: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
	dnsLabelValidation = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
)

const (
	defaultGRPCPort          int32 = 14250
	defaultThriftHTTPPort    int32 = 14268
	defaultThriftCompactPort int32 = 6831
	defaultThriftBinaryPort  int32 = 6832
)

// ConfigToReceiverPorts converts the incoming configuration object into a set of service ports required by the receivers.
func ConfigToReceiverPorts(logger logr.Logger, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	// now, we gather which ports we might need to open
	// for that, we get all the receivers and check their `endpoint` properties,
	// extracting the port from it. The port name has to be a "DNS_LABEL", so, we try to make it follow the pattern:
	// ${instance.Name}-${receiver.name}-${receiver.qualifier}
	// the receiver-name is typically the node name from the receivers map
	// the receiver-qualifier is what comes after the slash in the receiver name, but typically nil
	// examples:
	// ```yaml
	// receivers:
	//   examplereceiver:
	//     endpoint: 0.0.0.0:12345
	//   examplereceiver/settings:
	//     endpoint: 0.0.0.0:12346
	// in this case, we have two ports, named: "examplereceiver" and "examplereceiver-settings"
	receiversProperty, ok := config["receivers"]
	if !ok {
		return nil, ErrNoReceivers
	}

	receivers, ok := receiversProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrReceiversNotAMap
	}

	ports := []corev1.ServicePort{}
	for key, val := range receivers {
		receiver, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.Info("receiver doesn't seem to be a map of properties")
			continue
		}

		// Jaeger has multiple protocols, each on its own endpoint, so, we parse it separately
		if strings.HasPrefix(key.(string), "jaeger") {
			jaegerPorts := parseJaegerEndpoints(logger, key.(string), receiver)
			if len(jaegerPorts) > 0 {
				ports = append(ports, jaegerPorts...)
			}
			continue
		}

		port := parseGenericReceiverEndpoint(logger, key.(string), receiver)
		if port != nil {
			ports = append(ports, *port)
		}
	}

	return ports, nil
}

func parseJaegerEndpoints(logger logr.Logger, name string, receiver map[interface{}]interface{}) []corev1.ServicePort {
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
		if receiverProtocol, ok := receiver[protocol.name]; ok {
			// we have the specified protocol, we definitely need a service port
			nameWithProtocol := fmt.Sprintf("%s-%s", name, protocol.name)
			var protocolPort *corev1.ServicePort

			// do we have a configuration block for the protocol?
			settings, ok := receiverProtocol.(map[interface{}]interface{})
			if ok {
				protocolPort = parseGenericReceiverEndpoint(logger, nameWithProtocol, settings)
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

	return ports
}

func parseGenericReceiverEndpoint(logger logr.Logger, name string, receiver map[interface{}]interface{}) *corev1.ServicePort {
	endpoint, ok := receiver["endpoint"]
	if !ok {
		logger.Info("receiver doesn't have an endpoint")
		return nil
	}

	switch endpoint := endpoint.(type) {
	case string:
		port, err := portFromEndpoint(endpoint)
		if err != nil {
			logger.WithValues("endpoint", endpoint).Info("couldn't parse the endpoint's port")
			return nil
		}

		return &corev1.ServicePort{
			Name: portName(name, port),
			Port: port,
		}
	default:
		logger.Info("receiver's endpoint isn't a string")
	}

	return nil
}

func portName(receiverName string, port int32) string {
	if len(receiverName) > 63 {
		return fmt.Sprintf("port-%d", port)
	}

	candidate := strings.ReplaceAll(receiverName, "/", "-")
	candidate = strings.ReplaceAll(candidate, "_", "-")

	if !dnsLabelValidation.MatchString(candidate) {
		return fmt.Sprintf("port-%d", port)
	}

	// matches the pattern and has less than 63 chars -- the candidate name is good to go!
	return candidate
}

func portFromEndpoint(endpoint string) (int32, error) {
	i := strings.LastIndex(endpoint, ":") + 1
	part := endpoint[i:]
	port, err := strconv.Atoi(part)
	return int32(port), err
}
