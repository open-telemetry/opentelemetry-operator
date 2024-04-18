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

import corev1 "k8s.io/api/core/v1"

var (
	genericReceivers = map[string][]Option{
		"awsxray": {
			WithDefaultPort(2000),
		},
		"carbon": {
			WithDefaultPort(2003),
		},
		"collectd": {
			WithDefaultPort(8081),
		},
		"fluentforward": {
			WithDefaultPort(8006),
		},
		"influxdb": {
			WithDefaultPort(8086),
		},
		"opencensus": {
			WithDefaultPort(55678),
			WithDefaultAppProtocol(nil),
		},
		"sapm": {
			WithDefaultPort(7276),
		},
		"signalfx": {
			WithDefaultPort(9943),
		},
		"splunk_hec": {
			WithDefaultPort(8088),
		},
		"statsd": {
			WithDefaultPort(8125),
			WithDefaultProtocol(corev1.ProtocolUDP),
		},
		"tcplog": {
			WithDefaultPort(0),
			WithDefaultProtocol(corev1.ProtocolTCP),
		},
		"udplog": {
			WithDefaultPort(0),
			WithDefaultProtocol(corev1.ProtocolUDP),
		},
		"wavefront": {
			WithDefaultPort(2003),
		},
		"zipkin": {
			WithDefaultPort(9411),
			WithDefaultAppProtocol(&http),
		},
	}
	genericMultiPortReceivers = map[string][]MultiPortOption{
		"skywalking": {
			WithPortMapping(grpc, 11800,
				WithTargetPort(11800),
				WithAppProtocol(&grpc),
			),
			WithPortMapping(http, 12800,
				WithTargetPort(12800),
				WithAppProtocol(&http),
			),
		},
		"jaeger": {
			WithPortMapping(grpc, 14250,
				WithProtocol(corev1.ProtocolTCP),
				WithAppProtocol(&grpc),
			),
			WithPortMapping("thrift_http", 14268,
				WithProtocol(corev1.ProtocolTCP),
				WithAppProtocol(&http),
			),
			WithPortMapping("thrift_compact", 6831,
				WithProtocol(corev1.ProtocolUDP),
			),
			WithPortMapping("thrift_binary", 6832,
				WithProtocol(corev1.ProtocolUDP),
			),
		},
		"loki": {
			WithPortMapping(grpc, 9095,
				WithTargetPort(9095),
				WithAppProtocol(&grpc),
			),
			WithPortMapping(http, 3100,
				WithTargetPort(3100),
				WithAppProtocol(&http),
			),
		},
	}
)

func init() {
	for name, options := range genericReceivers {
		Register(name, createParser(options...))
	}
	for name, options := range genericMultiPortReceivers {
		Register(name, createMultiPortParser(options...))
	}
}
