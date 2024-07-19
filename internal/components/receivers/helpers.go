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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// registry holds a record of all known receiver parsers.
var registry = make(map[string]components.ComponentParser)

// Register adds a new parser builder to the list of known builders.
func Register(name string, p components.ComponentParser) {
	registry[name] = p
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[components.ComponentType(name)]
	return ok
}

// ReceiverFor returns a parser builder for the given exporter name.
func ReceiverFor(name string) components.ComponentParser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	return components.NewComponentParser(
		components.WithComponentPortParser(
			components.NewSilentSinglePortParser(components.ComponentType(name), components.UnsetPort),
		),
	)
}

var (
	componentParsers = []components.ComponentParser{
		components.NewComponentParser(
			// TODO: components.WithDefaultConfigOTLP(),
			components.WithComponentPortParser(
				components.NewMultiPortReceiver("otlp",
					components.WithPortMapping(
						"grpc",
						4317,
						components.WithAppProtocol(&components.GrpcProtocol),
						components.WithTargetPort(4317),
					), components.WithPortMapping(
						"http",
						4318,
						components.WithAppProtocol(&components.HttpProtocol),
						components.WithTargetPort(4318),
					),
				),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewMultiPortReceiver("skywalking",
					components.WithPortMapping(components.GrpcProtocol, 11800,
						components.WithTargetPort(11800),
						components.WithAppProtocol(&components.GrpcProtocol),
					),
					components.WithPortMapping(components.HttpProtocol, 12800,
						components.WithTargetPort(12800),
						components.WithAppProtocol(&components.HttpProtocol),
					),
				),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewMultiPortReceiver("jaeger",
					components.WithPortMapping(components.GrpcProtocol, 14250,
						components.WithTargetPort(14250),
						components.WithProtocol(corev1.ProtocolTCP),
						components.WithAppProtocol(&components.GrpcProtocol),
					),
					components.WithPortMapping("thrift_http", 14268,
						components.WithTargetPort(14268),
						components.WithProtocol(corev1.ProtocolTCP),
						components.WithAppProtocol(&components.HttpProtocol),
					),
					components.WithPortMapping("thrift_compact", 6831,
						components.WithTargetPort(6831),
						components.WithProtocol(corev1.ProtocolUDP),
					),
					components.WithPortMapping("thrift_binary", 6832,
						components.WithTargetPort(6832),
						components.WithProtocol(corev1.ProtocolUDP),
					),
				),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewMultiPortReceiver("loki",
					components.WithPortMapping(components.GrpcProtocol, 9095,
						components.WithTargetPort(9095),
						components.WithAppProtocol(&components.GrpcProtocol),
					),
					components.WithPortMapping(components.HttpProtocol, 3100,
						components.WithTargetPort(3100),
						components.WithAppProtocol(&components.HttpProtocol),
					),
				),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("awsxray", 2000, components.WithTargetPort(2000)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("carbon", 2003, components.WithTargetPort(2003)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("collectd", 8081, components.WithTargetPort(8081)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("fluentforward", 8006, components.WithTargetPort(8006)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("influxdb", 8086, components.WithTargetPort(8086)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("opencensus", 55678, components.WithAppProtocol(nil), components.WithTargetPort(55678)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("sapm", 7276, components.WithTargetPort(7276)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("signalfx", 9943, components.WithTargetPort(9943)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("splunk_hec", 8088, components.WithTargetPort(8088)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("statsd", 8125, components.WithProtocol(corev1.ProtocolUDP), components.WithTargetPort(8125)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("tcplog", components.UnsetPort, components.WithProtocol(corev1.ProtocolTCP)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("udplog", components.UnsetPort, components.WithProtocol(corev1.ProtocolUDP)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("wavefront", 2003, components.WithTargetPort(2003)),
			),
		),
		components.NewComponentParser(
			components.WithComponentPortParser(
				components.NewSinglePortParser("zipkin", 9411, components.WithAppProtocol(&components.HttpProtocol), components.WithProtocol(corev1.ProtocolTCP), components.WithTargetPort(3100)),
			),
		),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("prometheus"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("kubeletstats"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("sshcheck"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("cloudfoundry"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("vcenter"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("oracledb"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("snmp"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("googlecloudpubsub"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("chrony"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("jmx"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("podman_stats"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("pulsar"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("docker_stats"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("aerospike"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("zookeeper"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("prometheus_simple"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("saphana"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("riak"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("redis"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("rabbitmq"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("purefb"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("postgresql"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("nsxt"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("nginx"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("mysql"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("memcached"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("httpcheck"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("haproxy"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("flinkmetrics"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("couchdb"))),
		components.NewComponentParser(components.WithComponentPortParser(NewScraperParser("filelog"))),
	}
)

func init() {
	for _, parser := range componentParsers {
		Register(parser.ParserType(), parser)
	}
}
