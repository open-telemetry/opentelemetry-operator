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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers/receivers"
)

var logger = logf.Log.WithName("unit-tests")

func TestParseEndpoint(t *testing.T) {
	// prepare
	// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
	builder, err := receivers.NewGenericReceiverParser("myreceiver", map[string]interface{}{
		"endpoint": "0.0.0.0:1234",
	})
	assert.NoError(t, err)

	// test
	ports, err := builder.Ports(logger)

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 1234, ports[0].Port)
}

func TestFailedToParseEndpoint(t *testing.T) {
	// prepare
	// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
	builder, err := receivers.NewGenericReceiverParser("myreceiver", map[string]interface{}{
		"endpoint": "0.0.0.0",
	})
	assert.NoError(t, err)

	// test
	ports, err := builder.Ports(logger)

	// verify
	assert.Error(t, err)
	assert.Len(t, ports, 0)
}

func TestDownstreamParsers(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		receiverName string
		parserName   string
		defaultPort  int
	}{
		{"zipkin", "zipkin", "__zipkin", 9411},
		{"opencensus", "opencensus", "__opencensus", 55678},

		// contrib receivers
		{"carbon", "carbon", "__carbon", 2003},
		{"collectd", "collectd", "__collectd", 8081},
		{"sapm", "sapm", "__sapm", 7276},
		{"signalfx", "signalfx", "__signalfx", 9943},
		{"wavefront", "wavefront", "__wavefront", 2003},
		{"fluentforward", "fluentforward", "__fluentforward", 8006},
		{"statsd", "statsd", "__statsd", 8125},
		{"influxdb", "influxdb", "__influxdb", 8086},
		{"splunk_hec", "splunk_hec", "__splunk_hec", 8088},
		{"awsxray", "awsxray", "__awsxray", 2000},
	} {
		t.Run(tt.receiverName, func(t *testing.T) {
			t.Run("builds successfully", func(t *testing.T) {
				// test
				builder, err := receivers.For(tt.receiverName, map[string]interface{}{})
				assert.NoError(t, err)

				// verify
				assert.Equal(t, tt.parserName, builder.ParserName())
			})

			t.Run("assigns the expected port", func(t *testing.T) {
				// prepare
				builder, err := receivers.For(tt.receiverName, map[string]interface{}{})
				assert.NoError(t, err)

				// test
				ports, err := builder.Ports(logger)

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, tt.defaultPort, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.receiverName, int32(tt.defaultPort)), ports[0].Name)
			})

			t.Run("allows port to be overridden", func(t *testing.T) {
				// prepare
				builder, err := receivers.For(tt.receiverName, map[string]interface{}{
					"endpoint": "0.0.0.0:65535",
				})
				assert.NoError(t, err)

				// test
				ports, err := builder.Ports(logger)

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 1)
				assert.EqualValues(t, 65535, ports[0].Port)
				assert.Equal(t, naming.PortName(tt.receiverName, int32(tt.defaultPort)), ports[0].Name)
			})
		})
	}
}
