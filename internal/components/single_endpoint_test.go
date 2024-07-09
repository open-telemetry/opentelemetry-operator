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
		name string
		port int32
		opts []components.PortBuilderOption
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				name: "receiver1",
				opts: nil,
			},
			want: "__receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				name: "receiver2",
				opts: []components.PortBuilderOption{
					components.WithTargetPort(8080),
				},
			},
			want: "__receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := components.NewSinglePortParser(tt.fields.name, tt.fields.port, tt.fields.opts...)
			assert.Equalf(t, tt.want, s.ParserName(), "ParserName()")
		})
	}
}

func TestSingleEndpointParser_ParserType(t *testing.T) {
	type fields struct {
		name string
		port int32
		opts []components.PortBuilderOption
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				name: "receiver1",
				opts: nil,
			},
			want: "receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				name: "receiver2/test",
				opts: []components.PortBuilderOption{
					components.WithTargetPort(80),
				},
			},
			want: "receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := components.NewSinglePortParser(tt.fields.name, tt.fields.port, tt.fields.opts...)
			assert.Equalf(t, tt.want, s.ParserType(), "ParserType()")
		})
	}
}

func TestSingleEndpointParser_Ports(t *testing.T) {
	type fields struct {
		name string
		port int32
		opts []components.PortBuilderOption
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
				name: "testparser",
				port: 8080,
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
				name: "testparser",
				port: 8080,
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
				name: "testparser",
				port: 8080,
				opts: []components.PortBuilderOption{
					components.WithTargetPort(4317),
					components.WithProtocol(corev1.ProtocolTCP),
					components.WithAppProtocol(&components.GrpcProtocol),
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{
					Name:        "testparser",
					Port:        8080,
					TargetPort:  intstr.FromInt32(4317),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "InvalidConfigMissingPort",
			fields: fields{
				name: "testparser",
				port: 0,
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
				name: "testparser",
				port: 8080,
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
			s := components.NewSinglePortParser(tt.fields.name, tt.fields.port, tt.fields.opts...)
			got, err := s.Ports(logr.Discard(), tt.fields.name, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
		})
	}
}

func TestNewSilentSinglePortParser_Ports(t *testing.T) {
	type fields struct {
		name string
		port int32
		opts []components.PortBuilderOption
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
				name: "testparser",
				port: 8080,
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
				name: "testparser",
				port: 8080,
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
				name: "testparser",
				port: 8080,
				opts: []components.PortBuilderOption{
					components.WithTargetPort(4317),
					components.WithProtocol(corev1.ProtocolTCP),
					components.WithAppProtocol(&components.GrpcProtocol),
				},
			},
			args: args{
				config: map[string]interface{}{},
			},
			want: []corev1.ServicePort{
				{
					Name:        "testparser",
					Port:        8080,
					TargetPort:  intstr.FromInt32(4317),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "InvalidConfigMissingPort",
			fields: fields{
				name: "testparser",
				port: 0,
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
				name: "testparser",
				port: 8080,
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
			s := components.NewSilentSinglePortParser(tt.fields.name, tt.fields.port, tt.fields.opts...)
			got, err := s.Ports(logr.Discard(), tt.fields.name, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
		})
	}
}
