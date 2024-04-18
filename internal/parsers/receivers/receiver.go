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

package receivers

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

type endpointContainer struct {
	Endpoint string `json:"endpoint,omitempty"`
}

func (ec *endpointContainer) getPortNum() (int32, error) {
	return portFromEndpoint(ec.Endpoint)
}

func (ec *endpointContainer) getPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := ec.getPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default for http: %d", p)
		return p
	}
	return num
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
	portNotFoundErr = errors.New("port should not be empty")
)

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

// registry holds a record of all known receiver parsers.
var registry = make(map[string]parsers.Builder)

// BuilderFor returns a parser builder for the given receiver name.
func BuilderFor(name string) parsers.Builder {
	builder := registry[receiverType(name)]
	if builder == nil {
		builder = NewGenericReceiverParser
	}

	return builder
}

// For returns a new parser for the given receiver name + config.
func For(name string, config interface{}) (parsers.ComponentPortParser, error) {
	return BuilderFor(name)(name, config)
}

// Register adds a new parser builder to the list of known builders.
func Register(name string, builder parsers.Builder) {
	registry[name] = builder
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}
