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

package receiver

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
)

const (
	unsetPort = 0
)

var (
	grpc             = "grpc"
	http             = "http"
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
	genericReceivers = map[string][]parser.SinglePortOption{
		"awsxray": {
			parser.WithSinglePort(2000),
		},
		"carbon": {
			parser.WithSinglePort(2003),
		},
		"collectd": {
			parser.WithSinglePort(8081),
		},
		"fluentforward": {
			parser.WithSinglePort(8006),
		},
		"influxdb": {
			parser.WithSinglePort(8086),
		},
		"opencensus": {
			parser.WithSinglePort(55678, parser.WithAppProtocol(nil)),
		},
		"sapm": {
			parser.WithSinglePort(7276),
		},
		"signalfx": {
			parser.WithSinglePort(9943),
		},
		"splunk_hec": {
			parser.WithSinglePort(8088),
		},
		"statsd": {
			parser.WithSinglePort(8125, parser.WithProtocol(corev1.ProtocolUDP)),
		},
		"tcplog": {
			parser.WithSinglePort(unsetPort, parser.WithProtocol(corev1.ProtocolTCP)),
		},
		"udplog": {
			parser.WithSinglePort(unsetPort, parser.WithProtocol(corev1.ProtocolUDP)),
		},
		"wavefront": {
			parser.WithSinglePort(2003),
		},
		"zipkin": {
			parser.WithSinglePort(9411, parser.WithAppProtocol(&http), parser.WithProtocol(corev1.ProtocolTCP)),
		},
	}
	genericMultiPortReceivers = map[string][]MultiPortOption{
		"otlp": {
			WithPortMapping(
				"grpc",
				4317,
				parser.WithAppProtocol(&grpc),
				parser.WithTargetPort(4317),
			), WithPortMapping(
				"http",
				4318,
				parser.WithAppProtocol(&http),
				parser.WithTargetPort(4318),
			),
		},
		"skywalking": {
			WithPortMapping(grpc, 11800,
				parser.WithTargetPort(11800),
				parser.WithAppProtocol(&grpc),
			),
			WithPortMapping(http, 12800,
				parser.WithTargetPort(12800),
				parser.WithAppProtocol(&http),
			),
		},
		"jaeger": {
			WithPortMapping(grpc, 14250,
				parser.WithProtocol(corev1.ProtocolTCP),
				parser.WithAppProtocol(&grpc),
			),
			WithPortMapping("thrift_http", 14268,
				parser.WithProtocol(corev1.ProtocolTCP),
				parser.WithAppProtocol(&http),
			),
			WithPortMapping("thrift_compact", 6831,
				parser.WithProtocol(corev1.ProtocolUDP),
			),
			WithPortMapping("thrift_binary", 6832,
				parser.WithProtocol(corev1.ProtocolUDP),
			),
		},
		"loki": {
			WithPortMapping(grpc, 9095,
				parser.WithTargetPort(9095),
				parser.WithAppProtocol(&grpc),
			),
			WithPortMapping(http, 3100,
				parser.WithTargetPort(3100),
				parser.WithAppProtocol(&http),
			),
		},
	}
)

func init() {
	for name := range scraperReceivers {
		Register(name, NewScraperParser)
	}
	for name, options := range genericReceivers {
		Register(name, parser.CreateParser(options...))
	}
	for name, options := range genericMultiPortReceivers {
		Register(name, createMultiPortParser(multiProtocolEndpointConfigFactory, options...))
	}
}
