// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// registry holds a record of all known receiver parsers.
var registry = make(map[string]components.Parser)

// Register adds a new parser builder to the list of known builders.
func Register(name string, p components.Parser) {
	registry[name] = p
}

// IsRegistered checks whether a parser is registered with the given name.
func IsRegistered(name string) bool {
	_, ok := registry[components.ComponentType(name)]
	return ok
}

// ReceiverFor returns a parser builder for the given exporter name.
func ReceiverFor(name string) components.Parser {
	if parser, ok := registry[components.ComponentType(name)]; ok {
		return parser
	}
	return components.NewSilentSinglePortParserBuilder(components.ComponentType(name), components.UnsetPort).MustBuild()
}

// NewScraperParser is an instance of a generic parser that returns nothing when called and never fails.
func NewScraperParser(name string) *components.GenericParser[any] {
	return components.NewBuilder[any]().WithName(name).WithPort(components.UnsetPort).MustBuild()
}

var (
	componentParsers = []components.Parser{
		components.NewMultiPortReceiverBuilder("otlp").
			AddPortMapping(components.NewProtocolBuilder("grpc", 4317).
				WithAppProtocol(&components.GrpcProtocol).
				WithTargetPort(4317)).
			AddPortMapping(components.NewProtocolBuilder("http", 4318).
				WithAppProtocol(&components.HttpProtocol).
				WithTargetPort(4318)).
			MustBuild(),
		components.NewMultiPortReceiverBuilder("skywalking").
			AddPortMapping(components.NewProtocolBuilder(components.GrpcProtocol, 11800).
				WithTargetPort(11800).
				WithAppProtocol(&components.GrpcProtocol)).
			AddPortMapping(components.NewProtocolBuilder(components.HttpProtocol, 12800).
				WithTargetPort(12800).
				WithAppProtocol(&components.HttpProtocol)).
			MustBuild(),
		components.NewMultiPortReceiverBuilder("jaeger").
			AddPortMapping(components.NewProtocolBuilder(components.GrpcProtocol, 14250).
				WithTargetPort(14250).
				WithProtocol(corev1.ProtocolTCP).
				WithAppProtocol(&components.GrpcProtocol)).
			AddPortMapping(components.NewProtocolBuilder("thrift_http", 14268).
				WithTargetPort(14268).
				WithProtocol(corev1.ProtocolTCP).
				WithAppProtocol(&components.HttpProtocol)).
			AddPortMapping(components.NewProtocolBuilder("thrift_compact", 6831).
				WithTargetPort(6831).
				WithProtocol(corev1.ProtocolUDP)).
			AddPortMapping(components.NewProtocolBuilder("thrift_binary", 6832).
				WithTargetPort(6832).
				WithProtocol(corev1.ProtocolUDP)).
			MustBuild(),
		components.NewMultiPortReceiverBuilder("loki").
			AddPortMapping(components.NewProtocolBuilder(components.GrpcProtocol, 9095).
				WithTargetPort(9095).
				WithAppProtocol(&components.GrpcProtocol)).
			AddPortMapping(components.NewProtocolBuilder(components.HttpProtocol, 3100).
				WithTargetPort(3100).
				WithAppProtocol(&components.HttpProtocol)).
			MustBuild(),
		components.NewSinglePortParserBuilder("awsxray", 2000).
			WithTargetPort(2000).
			WithProtocol(corev1.ProtocolUDP).
			MustBuild(),
		components.NewSinglePortParserBuilder("carbon", 2003).
			WithTargetPort(2003).
			MustBuild(),
		components.NewSinglePortParserBuilder("collectd", 8081).
			WithTargetPort(8081).
			MustBuild(),
		components.NewSinglePortParserBuilder("fluentforward", 8006).
			WithTargetPort(8006).
			MustBuild(),
		components.NewSinglePortParserBuilder("influxdb", 8086).
			WithTargetPort(8086).
			MustBuild(),
		components.NewSinglePortParserBuilder("opencensus", 55678).
			WithAppProtocol(nil).
			WithTargetPort(55678).
			MustBuild(),
		components.NewSinglePortParserBuilder("sapm", 7276).
			WithTargetPort(7276).
			MustBuild(),
		components.NewSinglePortParserBuilder("signalfx", 9943).
			WithTargetPort(9943).
			MustBuild(),
		components.NewSinglePortParserBuilder("splunk_hec", 8088).
			WithTargetPort(8088).
			MustBuild(),
		components.NewSinglePortParserBuilder("statsd", 8125).
			WithProtocol(corev1.ProtocolUDP).
			WithTargetPort(8125).
			MustBuild(),
		components.NewSinglePortParserBuilder("tcplog", components.UnsetPort).
			WithProtocol(corev1.ProtocolTCP).
			MustBuild(),
		components.NewSinglePortParserBuilder("udplog", components.UnsetPort).
			WithProtocol(corev1.ProtocolUDP).
			MustBuild(),
		components.NewSinglePortParserBuilder("wavefront", 2003).
			WithTargetPort(2003).
			MustBuild(),
		components.NewSinglePortParserBuilder("zipkin", 9411).
			WithAppProtocol(&components.HttpProtocol).
			WithProtocol(corev1.ProtocolTCP).
			WithTargetPort(3100).
			MustBuild(),
		components.NewBuilder[kubeletStatsConfig]().WithName("kubeletstats").
			WithClusterRoleRulesGen(generateKubeletStatsClusterRoleRules).
			WithEnvVarGen(generateKubeletStatsEnvVars).
			MustBuild(),
		components.NewBuilder[k8seventsConfig]().WithName("k8s_events").
			WithClusterRoleRulesGen(generatek8seventsClusterRoleRules).
			MustBuild(),
		components.NewBuilder[k8sclusterConfig]().WithName("k8s_cluster").
			WithClusterRoleRulesGen(generatek8sclusterRbacRules).
			MustBuild(),
		components.NewBuilder[k8sobjectsConfig]().WithName("k8sobjects").
			WithClusterRoleRulesGen(generatek8sobjectsClusterRoleRules).
			MustBuild(),
		components.NewBuilder[prometheusReceiverConfig]().WithName("prometheus").
			WithPort(components.UnsetPort).
			WithRoleGen(generatePrometheusReceiverRoles).
			WithRoleBindingGen(generatePrometheusReceiverRoleBindings).
			MustBuild(),
		NewScraperParser("sshcheck"),
		NewScraperParser("cloudfoundry"),
		NewScraperParser("vcenter"),
		NewScraperParser("oracledb"),
		NewScraperParser("snmp"),
		NewScraperParser("googlecloudpubsub"),
		NewScraperParser("chrony"),
		NewScraperParser("jmx"),
		NewScraperParser("podman_stats"),
		NewScraperParser("pulsar"),
		NewScraperParser("docker_stats"),
		NewScraperParser("aerospike"),
		NewScraperParser("zookeeper"),
		NewScraperParser("prometheus_simple"),
		NewScraperParser("saphana"),
		NewScraperParser("riak"),
		NewScraperParser("redis"),
		NewScraperParser("rabbitmq"),
		NewScraperParser("purefb"),
		NewScraperParser("postgresql"),
		NewScraperParser("nsxt"),
		NewScraperParser("nginx"),
		NewScraperParser("mysql"),
		NewScraperParser("memcached"),
		NewScraperParser("httpcheck"),
		NewScraperParser("haproxy"),
		NewScraperParser("flinkmetrics"),
		NewScraperParser("couchdb"),
		NewScraperParser("filelog"),
	}
)

func init() {
	for _, parser := range componentParsers {
		Register(parser.ParserType(), parser)
	}
}
