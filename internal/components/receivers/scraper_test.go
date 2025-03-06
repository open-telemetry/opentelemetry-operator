// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

func TestScraperParsers(t *testing.T) {
	for _, tt := range []struct {
		receiverName string
		parserName   string
		defaultPort  int
	}{
		{"prometheus", "__prometheus", 0},
		{"kubeletstats", "__kubeletstats", 0},
		{"sshcheck", "__sshcheck", 0},
		{"cloudfoundry", "__cloudfoundry", 0},
		{"vcenter", "__vcenter", 0},
		{"oracledb", "__oracledb", 0},
		{"snmp", "__snmp", 0},
		{"googlecloudpubsub", "__googlecloudpubsub", 0},
		{"chrony", "__chrony", 0},
		{"jmx", "__jmx", 0},
		{"podman_stats", "__podman_stats", 0},
		{"pulsar", "__pulsar", 0},
		{"docker_stats", "__docker_stats", 0},
		{"aerospike", "__aerospike", 0},
		{"zookeeper", "__zookeeper", 0},
		{"prometheus_simple", "__prometheus_simple", 0},
		{"saphana", "__saphana", 0},
		{"riak", "__riak", 0},
		{"redis", "__redis", 0},
		{"rabbitmq", "__rabbitmq", 0},
		{"purefb", "__purefb", 0},
		{"postgresql", "__postgresql", 0},
		{"nsxt", "__nsxt", 0},
		{"nginx", "__nginx", 0},
		{"mysql", "__mysql", 0},
		{"memcached", "__memcached", 0},
		{"httpcheck", "__httpcheck", 0},
		{"haproxy", "__haproxy", 0},
		{"flinkmetrics", "__flinkmetrics", 0},
		{"couchdb", "__couchdb", 0},
	} {
		t.Run(tt.receiverName, func(t *testing.T) {
			t.Run("builds successfully", func(t *testing.T) {
				// test
				parser := receivers.ReceiverFor(tt.receiverName)

				// verify
				assert.Equal(t, tt.parserName, parser.ParserName())
			})

			t.Run("default is nothing", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				ports, err := parser.Ports(logger, tt.receiverName, map[string]interface{}{})

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 0)
			})

			t.Run("always returns nothing", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				ports, err := parser.Ports(logger, tt.receiverName, map[string]interface{}{
					"endpoint": "0.0.0.0:65535",
				})

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 0)
			})
		})
	}
}
