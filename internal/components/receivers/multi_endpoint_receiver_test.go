// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

var (
	grpc = "grpc"
	http = "http"
)

func TestMultiEndpointReceiverParsers(t *testing.T) {
	type testCase struct {
		name        string
		config      interface{}
		expectedErr error
		expectedSvc []corev1.ServicePort
	}
	type fields struct {
		receiverName string
		parserName   string
		cases        []testCase
	}
	for _, tt := range []fields{
		{
			receiverName: "jaeger",
			parserName:   "__jaeger",
			cases: []testCase{
				{
					name: "minimal config",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "jaeger-grpc",
							Port:        14250,
							TargetPort:  intstr.FromInt(14250),
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "grpc overridden",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{
								"endpoint": "0.0.0.0:1234",
							},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "jaeger-grpc",
							Port:        1234,
							TargetPort:  intstr.FromInt(1234),
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "all defaults",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc":           map[string]interface{}{},
							"thrift_http":    map[string]interface{}{},
							"thrift_compact": map[string]interface{}{},
							"thrift_binary":  map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "jaeger-grpc",
							Port:        14250,
							TargetPort:  intstr.FromInt(14250),
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &grpc,
						},
						{
							Name:        "port-14268",
							Port:        14268,
							TargetPort:  intstr.FromInt(14268),
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &http,
						},
						{
							Name:       "port-6831",
							Port:       6831,
							TargetPort: intstr.FromInt(6831),
							Protocol:   corev1.ProtocolUDP,
						},
						{
							Name:       "port-6832",
							Port:       6832,
							TargetPort: intstr.FromInt(6832),
							Protocol:   corev1.ProtocolUDP,
						},
					},
				},
			},
		},
		{
			receiverName: "otlp",
			parserName:   "__otlp",
			cases: []testCase{
				{
					name: "minimal config",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-grpc",
							Port:        4317,
							TargetPort:  intstr.FromInt32(4317),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "grpc overridden",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{
								"endpoint": "0.0.0.0:1234",
							},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-grpc",
							Port:        1234,
							TargetPort:  intstr.FromInt(1234),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "all defaults",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
							"http": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-grpc",
							Port:        4317,
							TargetPort:  intstr.FromInt32(4317),
							AppProtocol: &grpc,
						},
						{
							Name:        "otlp-http",
							Port:        4318,
							TargetPort:  intstr.FromInt32(4318),
							AppProtocol: &http,
						},
					},
				},
			},
		},
		{
			receiverName: "otlp/test",
			parserName:   "__otlp",
			cases: []testCase{
				{
					name: "minimal config",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-test-grpc",
							Port:        4317,
							TargetPort:  intstr.FromInt32(4317),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "grpc overridden",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{
								"endpoint": "0.0.0.0:1234",
							},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-test-grpc",
							Port:        1234,
							TargetPort:  intstr.FromInt32(1234),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "all defaults",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
							"http": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "otlp-test-grpc",
							Port:        4317,
							TargetPort:  intstr.FromInt32(4317),
							AppProtocol: &grpc,
						},
						{
							Name:        "otlp-test-http",
							Port:        4318,
							TargetPort:  intstr.FromInt32(4318),
							AppProtocol: &http,
						},
					},
				},
			},
		},
		{
			receiverName: "loki",
			parserName:   "__loki",
			cases: []testCase{
				{
					name: "minimal config",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "loki-grpc",
							Port:        9095,
							TargetPort:  intstr.FromInt32(9095),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "grpc overridden",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{
								"endpoint": "0.0.0.0:1234",
							},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "loki-grpc",
							Port:        1234,
							TargetPort:  intstr.FromInt(1234),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "all defaults",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
							"http": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "loki-grpc",
							Port:        9095,
							TargetPort:  intstr.FromInt32(9095),
							AppProtocol: &grpc,
						},
						{
							Name:        "loki-http",
							Port:        3100,
							TargetPort:  intstr.FromInt32(3100),
							AppProtocol: &http,
						},
					},
				},
			},
		},
		{
			receiverName: "skywalking",
			parserName:   "__skywalking",
			cases: []testCase{
				{
					name: "minimal config",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "skywalking-grpc",
							Port:        11800,
							TargetPort:  intstr.FromInt32(11800),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "grpc overridden",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{
								"endpoint": "0.0.0.0:1234",
							},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "skywalking-grpc",
							Port:        1234,
							TargetPort:  intstr.FromInt(1234),
							AppProtocol: &grpc,
						},
					},
				},
				{
					name: "all defaults",
					config: map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
							"http": map[string]interface{}{},
						},
					},
					expectedErr: nil,
					expectedSvc: []corev1.ServicePort{
						{
							Name:        "skywalking-grpc",
							Port:        11800,
							TargetPort:  intstr.FromInt32(11800),
							AppProtocol: &grpc,
						},
						{
							Name:        "skywalking-http",
							Port:        12800,
							TargetPort:  intstr.FromInt32(12800),
							AppProtocol: &http,
						},
					},
				},
			},
		},
	} {
		t.Run(tt.receiverName, func(t *testing.T) {
			t.Run("self registers", func(t *testing.T) {
				// verify
				assert.True(t, receivers.IsRegistered(tt.receiverName))
			})

			t.Run("is found by name", func(t *testing.T) {
				p := receivers.ReceiverFor(tt.receiverName)
				assert.Equal(t, tt.parserName, p.ParserName())
			})

			t.Run("bad config errors", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				_, err := parser.Ports(logger, tt.receiverName, []interface{}{"junk"})

				// verify
				assert.ErrorContains(t, err, "expected a map, got 'slice'")
			})
			t.Run("good config, unknown protocol", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				_, err := parser.Ports(logger, tt.receiverName, map[string]interface{}{
					"protocols": map[string]interface{}{
						"garbage": map[string]interface{}{},
					},
				})

				// verify
				assert.ErrorContains(t, err, "unknown protocol set: garbage")
			})
			for _, kase := range tt.cases {
				t.Run(kase.name, func(t *testing.T) {
					// prepare
					parser := receivers.ReceiverFor(tt.receiverName)

					// test
					ports, err := parser.Ports(logger, tt.receiverName, kase.config)
					if kase.expectedErr != nil {
						assert.EqualError(t, err, kase.expectedErr.Error())
						return
					}

					// verify
					assert.NoError(t, err)
					assert.Len(t, ports, len(kase.expectedSvc))
					assert.ElementsMatch(t, ports, kase.expectedSvc)
				})
			}

		})
	}
}
