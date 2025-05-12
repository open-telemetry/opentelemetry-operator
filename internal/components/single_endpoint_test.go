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
)

func TestSingleEndpointConfig_GetPortNumOrDefault(t *testing.T) {
	type fields struct {
		Endpoint      string
		ListenAddress string
	}
	type args struct {
		p int32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
	}{
		{
			name: "Test with valid endpoint",
			fields: fields{
				Endpoint:      "example.com:8080",
				ListenAddress: "",
			},
			args: args{
				p: 9000,
			},
			want: 8080,
		},
		{
			name: "Test with valid listen address",
			fields: fields{
				Endpoint:      "",
				ListenAddress: "0.0.0.0:9090",
			},
			args: args{
				p: 9000,
			},
			want: 9090,
		},
		{
			name: "Test with invalid configuration (no endpoint or listen address)",
			fields: fields{
				Endpoint:      "",
				ListenAddress: "",
			},
			args: args{
				p: 9000,
			},
			want: 9000, // Should return default port
		},
		{
			name: "Test with invalid endpoint format",
			fields: fields{
				Endpoint:      "invalid_endpoint",
				ListenAddress: "",
			},
			args: args{
				p: 9000,
			},
			want: 9000, // Should return default port
		},
		{
			name: "Test with invalid listen address format",
			fields: fields{
				Endpoint:      "",
				ListenAddress: "invalid_listen_address",
			},
			args: args{
				p: 9000,
			},
			want: 9000, // Should return default port
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &components.SingleEndpointConfig{
				Endpoint:      tt.fields.Endpoint,
				ListenAddress: tt.fields.ListenAddress,
			}
			assert.Equalf(t, tt.want, g.GetPortNumOrDefault(logr.Discard(), tt.args.p), "GetPortNumOrDefault(%v)", tt.args.p)
		})
	}
}

func TestSingleEndpointParser_ParserName(t *testing.T) {
	type fields struct {
		b components.Builder[*components.SingleEndpointConfig]
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				b: components.NewSinglePortParserBuilder("receiver1", components.UnsetPort),
			},
			want: "__receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				b: components.NewSinglePortParserBuilder("receiver2", components.UnsetPort).WithTargetPort(8080),
			},
			want: "__receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := tt.fields.b.Build()
			assert.NoError(t, err)
			assert.Equalf(t, tt.want, s.ParserName(), "ParserName()")
		})
	}
}

func TestSingleEndpointParser_ParserType(t *testing.T) {
	type fields struct {
		b components.Builder[*components.SingleEndpointConfig]
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				b: components.NewSinglePortParserBuilder("receiver1", components.UnsetPort),
			},
			want: "receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				b: components.NewSinglePortParserBuilder("receiver2", components.UnsetPort).WithTargetPort(8080),
			},
			want: "receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := tt.fields.b.Build()
			assert.NoError(t, err)
			assert.Equalf(t, tt.want, s.ParserType(), "ParserType()")
		})
	}
}

func TestSingleEndpointParser_Ports(t *testing.T) {
	type fields struct {
		b components.Builder[*components.SingleEndpointConfig]
	}
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []corev1.ServicePort
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ValidConfigWithPort",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: map[string]interface{}{
					"port": 8080,
				},
			},
			want: []corev1.ServicePort{
				{Name: "testparser", Port: 8080},
			},
			wantErr: assert.NoError,
		},
		{
			name: "ValidConfigWithPort nil config",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: nil,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "ValidConfigWithDefaultPort",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{Name: "testparser", Port: 8080},
			},
			wantErr: assert.NoError,
		},
		{
			name: "ConfigWithFixins",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", 8080).
					WithTargetPort(4317).
					WithProtocol(corev1.ProtocolTCP).
					WithAppProtocol(&components.GrpcProtocol),
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{
					Name:        "testparser",
					Port:        8080,
					TargetPort:  intstr.FromInt32(8080),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "InvalidConfigMissingPort",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", components.UnsetPort),
			},
			args: args{
				config: map[string]interface{}{
					"endpoint": "garbageeeee",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "ErrorParsingConfig",
			fields: fields{
				b: components.NewSinglePortParserBuilder("testparser", components.UnsetPort),
			},
			args: args{
				config: "invalid config",
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := tt.fields.b.Build()
			assert.NoError(t, err)
			got, err := s.Ports(logr.Discard(), s.ParserType(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
		})
	}
}

func TestNewSilentSinglePortParser_Ports(t *testing.T) {

	type fields struct {
		b components.Builder[*components.SingleEndpointConfig]
	}
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []corev1.ServicePort
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "ValidConfigWithPort",
			fields: fields{
				b: components.NewSilentSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: map[string]interface{}{
					"port": 8080,
				},
			},
			want: []corev1.ServicePort{
				{Name: "testparser", Port: 8080},
			},
			wantErr: assert.NoError,
		},
		{
			name: "ValidConfigWithDefaultPort",
			fields: fields{
				b: components.NewSilentSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{Name: "testparser", Port: 8080},
			},
			wantErr: assert.NoError,
		},
		{
			name: "ConfigWithFixins",
			fields: fields{
				b: components.NewSilentSinglePortParserBuilder("testparser", 8080).
					WithTargetPort(4317).
					WithProtocol(corev1.ProtocolTCP).
					WithAppProtocol(&components.GrpcProtocol),
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{
					Name:        "testparser",
					Port:        8080,
					TargetPort:  intstr.FromInt32(8080),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "InvalidConfigMissingPort",
			fields: fields{
				b: components.NewSilentSinglePortParserBuilder("testparser", components.UnsetPort),
			},
			args: args{
				config: map[string]interface{}{
					"endpoint": "garbageeeee",
				},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "ErrorParsingConfig",
			fields: fields{
				b: components.NewSilentSinglePortParserBuilder("testparser", 8080),
			},
			args: args{
				config: "invalid config",
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := tt.fields.b.Build()
			assert.NoError(t, err)
			got, err := s.Ports(logr.Discard(), s.ParserType(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
		})
	}
}
