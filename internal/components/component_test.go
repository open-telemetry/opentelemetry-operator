// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

func TestComponentType(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		name     string
		expected string
	}{
		{"regular case", "myreceiver", "myreceiver"},
		{"named instance", "myreceiver/custom", "myreceiver"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test and verify
			assert.Equal(t, tt.expected, components.ComponentType(tt.name))
		})
	}
}

func TestReceiverParsePortFromEndpoint(t *testing.T) {
	for _, tt := range []struct {
		desc          string
		endpoint      string
		expected      int
		errorExpected bool
	}{
		{"regular case", "http://localhost:1234", 1234, false},
		{"absolute with path", "http://localhost:1234/server-status?auto", 1234, false},
		{"no protocol", "0.0.0.0:1234", 1234, false},
		{"just port", ":1234", 1234, false},
		{"no port at all", "http://localhost", 0, true},
		{"overflow", "0.0.0.0:2147483648", 0, true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			val, err := components.PortFromEndpoint(tt.endpoint)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.EqualValues(t, tt.expected, val, "wrong port from endpoint %s: %d", tt.endpoint, val)
		})
	}
}

func TestGetPortsForConfig(t *testing.T) {
	type args struct {
		config    map[string]interface{}
		retriever components.ParserRetriever
	}
	tests := []struct {
		name    string
		args    args
		want    []corev1.ServicePort
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nothing",
			args: args{
				config:    nil,
				retriever: receivers.ReceiverFor,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "bad config",
			args: args{
				config: map[string]interface{}{
					"test": "garbage",
				},
				retriever: receivers.ReceiverFor,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "receivers",
			args: args{
				config: map[string]interface{}{
					"otlp": map[string]interface{}{
						"protocols": map[string]interface{}{
							"grpc": map[string]interface{}{},
						},
					},
				},
				retriever: receivers.ReceiverFor,
			},
			want: []corev1.ServicePort{
				{
					Name:        "otlp-grpc",
					Port:        4317,
					TargetPort:  intstr.FromInt32(4317),
					AppProtocol: &components.GrpcProtocol,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := components.GetPortsForConfig(logr.Discard(), tt.args.config, tt.args.retriever)
			if !tt.wantErr(t, err, fmt.Sprintf("GetPortsForConfig(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetPortsForConfig(%v)", tt.args.config)
		})
	}
}
