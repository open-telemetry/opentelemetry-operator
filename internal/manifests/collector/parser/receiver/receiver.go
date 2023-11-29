// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package parser is for parsing the OpenTelemetry Collector configuration.
package receiver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// registry holds a record of all known receiver parsers.
var registry = make(map[string]parser.Builder)

// BuilderFor returns a parser builder for the given receiver name.
func BuilderFor(name string) parser.Builder {
	builder := registry[receiverType(name)]
	if builder == nil {
		builder = NewGenericReceiverParser
	}

	return builder
}

// For returns a new parser for the given receiver name + config.
func For(logger logr.Logger, name string, config map[interface{}]interface{}) (parser.ComponentPortParser, error) {
	builder := BuilderFor(name)
	return builder(logger, name, config), nil
}

// Register adds a new parser builder to the list of known builders.
func Register(name string, builder parser.Builder) {
	registry[name] = builder
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

var (
	endpointKey      = "endpoint"
	listenAddressKey = "listen_address"
	scraperReceivers = map[string]struct{}{
		"prometheus":        {},
		"kubeletstats":      {},
		"sshcheck":          {},
		"cloudfoundry":      {},
		"vcenter":           {},
		"oracledb":          {},
		"snmp":              {},
		"googlecloudpubsub": {},
		"chrony":            {},
		"jmx":               {},
		"podman_stats":      {},
		"pulsar":            {},
		"docker_stats":      {},
		"aerospike":         {},
		"zookeeper":         {},
		"prometheus_simple": {},
		"saphana":           {},
		"riak":              {},
		"redis":             {},
		"rabbitmq":          {},
		"purefb":            {},
		"postgresql":        {},
		"nsxt":              {},
		"nginx":             {},
		"mysql":             {},
		"memcached":         {},
		"httpcheck":         {},
		"haproxy":           {},
		"flinkmetrics":      {},
		"couchdb":           {},
	}
)

func isScraperReceiver(name string) bool {
	_, exists := scraperReceivers[name]
	return exists
}

func singlePortFromConfigEndpoint(logger logr.Logger, name string, config map[interface{}]interface{}) *v1.ServicePort {
	var endpoint interface{}
	switch {
	// syslog receiver contains the endpoint
	// that needs to be exposed one level down inside config
	// i.e. either in tcp or udp section with field key
	// as `listen_address`
	case name == "syslog":
		var c map[interface{}]interface{}
		if udp, isUDP := config["udp"]; isUDP && udp != nil {
			c = udp.(map[interface{}]interface{})
			endpoint = getAddressFromConfig(logger, name, listenAddressKey, c)
		} else if tcp, isTCP := config["tcp"]; isTCP && tcp != nil {
			c = tcp.(map[interface{}]interface{})
			endpoint = getAddressFromConfig(logger, name, listenAddressKey, c)
		}

	// tcplog and udplog receivers hold the endpoint
	// value in `listen_address` field
	case name == "tcplog" || name == "udplog":
		endpoint = getAddressFromConfig(logger, name, listenAddressKey, config)

	// ignore the receiver as it holds the field key endpoint, and it
	// is a scraper, we only expose endpoint through k8s service objects for
	// receivers that aren't scrapers.
	case isScraperReceiver(name):
		return nil

	default:
		endpoint = getAddressFromConfig(logger, name, endpointKey, config)
	}

	switch e := endpoint.(type) {
	case nil:
		break
	case string:
		port, err := portFromEndpoint(e)
		if err != nil {
			logger.WithValues(endpointKey, e).Error(err, "couldn't parse the endpoint's port")
			return nil
		}

		return &corev1.ServicePort{
			Name: naming.PortName(name, port),
			Port: port,
		}
	default:
		logger.WithValues(endpointKey, endpoint).Error(fmt.Errorf("unrecognized type %T", endpoint), "receiver's endpoint isn't a string")
	}

	return nil
}

func getAddressFromConfig(logger logr.Logger, name, key string, config map[interface{}]interface{}) interface{} {
	endpoint, ok := config[key]
	if !ok {
		logger.V(2).Info("%s receiver doesn't have an %s", name, key)
		return nil
	}
	return endpoint
}

func portFromEndpoint(endpoint string) (int32, error) {
	var err error
	var port int64

	r := regexp.MustCompile(":[0-9]+")

	if r.MatchString(endpoint) {
		port, err = strconv.ParseInt(strings.Replace(r.FindString(endpoint), ":", "", -1), 10, 32)

		if err != nil {
			return 0, err
		}
	}

	if port == 0 {
		return 0, errors.New("port should not be empty")
	}

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
