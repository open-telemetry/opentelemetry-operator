package exporter

import (
	"reflect"
	"testing"

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
					Port: 9091,
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
			if got, _ := tt.parser.Ports(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ports(%v, = %v, want %v", tt.parser, got, tt.want)
			}
		})
	}
}
