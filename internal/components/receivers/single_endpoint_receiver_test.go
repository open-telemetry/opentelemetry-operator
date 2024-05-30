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

package receivers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestParseEndpoint(t *testing.T) {
	// prepare
	// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
	parser := receivers.BuilderFor("myreceiver")

	// test
	ports, err := parser.Ports(logger, map[string]interface{}{
		"endpoint": "0.0.0.0:1234",
	})

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 1234, ports[0].Port)
}

func TestFailedToParseEndpoint(t *testing.T) {
	// prepare
	// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
	parser := receivers.BuilderFor("myreceiver")

	// test
	ports, err := parser.Ports(logger, map[string]interface{}{
		"endpoint": "0.0.0.0",
	})

	// verify
	assert.Error(t, err)
	assert.Len(t, ports, 0)
}

func TestDownstreamParsers(t *testing.T) {
	for _, tt := range []struct {
		desc             string
		receiverName     string
		parserName       string
		defaultPort      int
		listenAddrParser bool
	}{
		{"zipkin", "zipkin", "__zipkin", 9411, false},
		{"opencensus", "opencensus", "__opencensus", 55678, false},

		// contrib receivers
		{"carbon", "carbon", "__carbon", 2003, false},
		{"collectd", "collectd", "__collectd", 8081, false},
		{"sapm", "sapm", "__sapm", 7276, false},
		{"signalfx", "signalfx", "__signalfx", 9943, false},
		{"wavefront", "wavefront", "__wavefront", 2003, false},
		{"fluentforward", "fluentforward", "__fluentforward", 8006, false},
		{"statsd", "statsd", "__statsd", 8125, false},
		{"influxdb", "influxdb", "__influxdb", 8086, false},
		{"splunk_hec", "splunk_hec", "__splunk_hec", 8088, false},
		{"awsxray", "awsxray", "__awsxray", 2000, false},
		{"tcplog", "tcplog", "__tcplog", 0, true},
		{"udplog", "udplog", "__udplog", 0, true},
	} {
		t.Run(tt.receiverName, func(t *testing.T) {
			t.Run("builds successfully", func(t *testing.T) {
				// test
				parser := receivers.BuilderFor(tt.receiverName)

				// verify
				assert.Equal(t, tt.parserName, parser.ParserName())
			})
			t.Run("bad config errors", func(t *testing.T) {
				// prepare
				parser := receivers.BuilderFor(tt.receiverName)

				// test throwing in pure junk
				_, err := parser.Ports(logger, func() {})

				// verify
				assert.ErrorContains(t, err, "expected a map, got 'func'")
			})

			t.Run("assigns the expected port", func(t *testing.T) {
				// prepare
				parser := receivers.BuilderFor(tt.receiverName)

				// test
				ports, err := parser.Ports(logger, map[string]interface{}{})

				if tt.defaultPort == 0 {
					assert.Len(t, ports, 0)
					return
				}
				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, tt.defaultPort, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.receiverName, int32(tt.defaultPort)), ports[0].Name)
			})

			t.Run("allows port to be overridden", func(t *testing.T) {
				// prepare
				parser := receivers.BuilderFor(tt.receiverName)

				// test
				var ports []corev1.ServicePort
				var err error
				if tt.listenAddrParser {
					ports, err = parser.Ports(logger, map[string]interface{}{
						"listen_address": "0.0.0.0:65535",
					})
				} else {
					ports, err = parser.Ports(logger, map[string]interface{}{
						"endpoint": "0.0.0.0:65535",
					})
				}

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, 65535, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.receiverName, int32(tt.defaultPort)), ports[0].Name)
			})
		})
	}
}
