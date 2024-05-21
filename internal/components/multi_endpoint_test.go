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

var (
	httpConfig = map[string]interface{}{
		"protocols": map[string]interface{}{
			"http": map[string]interface{}{},
		},
	}
	httpAndGrpcConfig = map[string]interface{}{
		"protocols": map[string]interface{}{
			"http": map[string]interface{}{},
			"grpc": map[string]interface{}{},
		},
	}
)

func TestMultiPortReceiver_ParserName(t *testing.T) {
	type fields struct {
		name string
		opts []components.MultiPortOption
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
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
				},
			},
			want: "__receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := components.NewMultiPortReceiver(tt.fields.name, tt.fields.opts...)
			assert.Equalf(t, tt.want, m.ParserName(), "ParserName()")
		})
	}
}

func TestMultiPortReceiver_ParserType(t *testing.T) {
	type fields struct {
		name string
		opts []components.MultiPortOption
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
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
				},
			},
			want: "receiver2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := components.NewMultiPortReceiver(tt.fields.name, tt.fields.opts...)
			assert.Equalf(t, tt.want, m.ParserType(), "ParserType()")
		})
	}
}

func TestMultiPortReceiver_Ports(t *testing.T) {
	type fields struct {
		name string
		opts []components.MultiPortOption
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
			name: "no options",
			fields: fields{
				name: "receiver1",
				opts: nil,
			},
			args: args{
				config: nil,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "single port mapping without builder options",
			fields: fields{
				name: "receiver2",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
				},
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name: "receiver2-http",
					Port: 80,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "port mapping with target port",
			fields: fields{
				name: "receiver3",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80, components.WithTargetPort(8080)),
				},
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:       "receiver3-http",
					Port:       80,
					TargetPort: intstr.FromInt32(8080),
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "port mapping with app protocol",
			fields: fields{
				name: "receiver4",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80, components.WithAppProtocol(&components.HttpProtocol)),
				},
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:        "receiver4-http",
					Port:        80,
					AppProtocol: &components.HttpProtocol,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "port mapping with protocol",
			fields: fields{
				name: "receiver5",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80, components.WithProtocol(corev1.ProtocolTCP)),
				},
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:     "receiver5-http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple port mappings",
			fields: fields{
				name: "receiver6",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
					components.WithPortMapping("grpc", 4317,
						components.WithTargetPort(4317),
						components.WithProtocol(corev1.ProtocolTCP),
						components.WithAppProtocol(&components.GrpcProtocol)),
				},
			},
			args: args{
				config: httpAndGrpcConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:        "receiver6-grpc",
					Port:        4317,
					TargetPort:  intstr.FromInt32(4317),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
				{
					Name: "receiver6-http",
					Port: 80,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple port mappings only one enabled",
			fields: fields{
				name: "receiver6",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
					components.WithPortMapping("grpc", 4317,
						components.WithTargetPort(4317),
						components.WithProtocol(corev1.ProtocolTCP),
						components.WithAppProtocol(&components.GrpcProtocol)),
				},
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name: "receiver6-http",
					Port: 80,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "error unmarshalling configuration",
			fields: fields{
				name: "receiver1",
				opts: nil,
			},
			args: args{
				config: "invalid config", // Simulate an invalid config that causes LoadMap to fail
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "error marshaling configuration",
			fields: fields{
				name: "receiver1",
				opts: nil,
			},
			args: args{
				config: func() {}, // Simulate an invalid config that causes LoadMap to fail
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "unknown protocol",
			fields: fields{
				name: "receiver2",
				opts: []components.MultiPortOption{
					components.WithPortMapping("http", 80),
				},
			},
			args: args{
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"unknown": map[string]interface{}{},
					},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := components.NewMultiPortReceiver(tt.fields.name, tt.fields.opts...)
			got, err := m.Ports(logr.Discard(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
		})
	}
}
