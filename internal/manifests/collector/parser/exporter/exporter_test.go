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

package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPorts(t *testing.T) {
	tests := []struct {
		testName string
		parser   *PrometheusExporterParser
		want     []v1.ServicePort
	}{
		{
			testName: "Valid Configuration",
			parser: &PrometheusExporterParser{
				name: "test-exporter",
				config: map[interface{}]interface{}{
					"endpoint": "http://myprometheus.io:9090",
				},
			},
			want: []v1.ServicePort{
				{
					Name: "test-exporter",
					Port: 9090,
				},
			},
		},
		{
			testName: "Empty Configuration",
			parser: &PrometheusExporterParser{
				name:   "test-exporter",
				config: nil, // Simulate no configuration provided
			},
			want: []v1.ServicePort{
				{
					Name:       "test-exporter",
					Port:       defaultPrometheusPort,
					TargetPort: intstr.FromInt(int(defaultPrometheusPort)),
					Protocol:   v1.ProtocolTCP,
				},
			},
		},
		{
			testName: "Invalid Endpoint No Port",
			parser: &PrometheusExporterParser{
				name: "test-exporter",
				config: map[interface{}]interface{}{
					"endpoint": "invalidendpoint",
				},
			},
			want: []v1.ServicePort{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			ports, _ := tt.parser.Ports()
			assert.Equal(t, tt.want, ports)
		})
	}
}
