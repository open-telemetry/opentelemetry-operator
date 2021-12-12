// Package parser is for parsing the OpenTelemetry Collector configuration.
package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

var (
	// DNS_LABEL constraints: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
	dnsLabelValidation = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
)

// ReceiverParser is an interface that should be implemented by all receiver parsers
type ReceiverParser interface {
	// Ports returns the service ports parsed based on the receiver's configuration
	Ports() ([]corev1.ServicePort, error)

	// ParserName returns the name of this parser
	ParserName() string
}

// Builder specifies the signature required for parser builders
type Builder func(logr.Logger, string, map[interface{}]interface{}) ReceiverParser

// registry holds a record of all known parsers
var registry = make(map[string]Builder)

// BuilderFor returns a parser builder for the given receiver name
func BuilderFor(name string) Builder {
	builder := registry[receiverType(name)]
	if builder == nil {
		builder = NewGenericReceiverParser
	}

	return builder
}

// For returns a new parser for the given receiver name + config
func For(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
	builder := BuilderFor(name)
	return builder(logger, name, config)
}

// Register adds a new parser builder to the list of known builders
func Register(name string, builder Builder) {
	registry[name] = builder
}

// IsRegistered checks whether a parser is registered with the given name
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

func singlePortFromConfigEndpoint(logger logr.Logger, name string, config map[interface{}]interface{}) *v1.ServicePort {
	endpoint, ok := config["endpoint"]
	if !ok {
		logger.V(2).Info("receiver doesn't have an endpoint")
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
	port, err := strconv.ParseInt(part, 10, 0)
	return int32(port), err
}

func receiverType(name string) string {
	// receivers have a name like:
	// - myreceiver/custom
	// - myreceiver
	// we extract the "myreceiver" part and see if we have a parser for the receiver
	if strings.Contains(name, "/") {
		return name[:strings.Index(name, "/")]
	}

	return name
}
