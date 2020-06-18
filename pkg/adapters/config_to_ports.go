package adapters

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/parser"
	corev1 "k8s.io/api/core/v1"
)

var (
	// ErrNoReceivers indicates that there are no receivers in the configuration
	ErrNoReceivers = errors.New("no receivers available as part of the configuration")

	// ErrReceiversNotAMap indicates that the receivers property isn't a map of values
	ErrReceiversNotAMap = errors.New("receivers property in the configuration doesn't contain valid receivers")
)

// ConfigToReceiverPorts converts the incoming configuration object into a set of service ports required by the receivers.
func ConfigToReceiverPorts(ctx context.Context, config map[interface{}]interface{}) ([]corev1.ServicePort, error) {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

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

		rcvrName := key.(string)
		rcvrParser := parser.For(rcvrName, receiver)

		rcvrPorts, err := rcvrParser.Ports(ctx)
		if err != nil {
			// should we break the process and return an error, or just ignore this faulty parser
			// and let the other parsers add their ports to the service? right now, the best
			// option seems to be to log the failures and move on, instead of failing them all
			logger.Error(err, "parser for '%s' has returned an error: %v", rcvrName, err)
			continue
		}

		if len(rcvrPorts) > 0 {
			ports = append(ports, rcvrPorts...)
		}
	}
	return ports, nil
}
