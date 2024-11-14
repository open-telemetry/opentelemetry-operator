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
		b components.MultiPortBuilder[*components.MultiProtocolEndpointConfig]
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				b: components.NewMultiPortReceiverBuilder("receiver1"),
			},
			want: "__receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				b: components.NewMultiPortReceiverBuilder("receiver2").AddPortMapping(
					components.NewProtocolBuilder("http", 80),
				),
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

func TestMultiPortReceiver_ParserType(t *testing.T) {
	type fields struct {
		b components.MultiPortBuilder[*components.MultiProtocolEndpointConfig]
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "no options",
			fields: fields{
				b: components.NewMultiPortReceiverBuilder("receiver1"),
			},
			want: "receiver1",
		},
		{
			name: "with port mapping without builder options",
			fields: fields{
				b: components.NewMultiPortReceiverBuilder("receiver2").AddPortMapping(
					components.NewProtocolBuilder("http", 80),
				),
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

func TestMultiPortReceiver_Ports(t *testing.T) {
	type fields struct {
		name string
		b    components.MultiPortBuilder[*components.MultiProtocolEndpointConfig]
	}
	type args struct {
		config interface{}
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		want         []corev1.ServicePort
		wantErr      assert.ErrorAssertionFunc
		wantBuildErr assert.ErrorAssertionFunc
	}{
		{
			name: "no options",
			fields: fields{
				name: "receiver1",
				b:    components.NewMultiPortReceiverBuilder("receiver1"),
			},
			args: args{
				config: nil,
			},
			want:         nil,
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "single port mapping without builder options",
			fields: fields{
				name: "receiver2",
				b:    components.NewMultiPortReceiverBuilder("receiver2").AddPortMapping(components.NewProtocolBuilder("http", 80)),
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
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "port mapping with target port",
			fields: fields{
				name: "receiver3",
				b: components.NewMultiPortReceiverBuilder("receiver3").
					AddPortMapping(components.NewProtocolBuilder("http", 80).
						WithTargetPort(8080)),
			},
			args: args{
				config: httpConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:       "receiver3-http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "port mapping with app protocol",
			fields: fields{
				name: "receiver4",
				b: components.NewMultiPortReceiverBuilder("receiver4").
					AddPortMapping(components.NewProtocolBuilder("http", 80).
						WithAppProtocol(&components.HttpProtocol)),
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
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "port mapping with protocol",
			fields: fields{
				name: "receiver5",
				b: components.NewMultiPortReceiverBuilder("receiver2").
					AddPortMapping(components.NewProtocolBuilder("http", 80).
						WithProtocol(corev1.ProtocolTCP)),
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
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "multiple port mappings",
			fields: fields{
				name: "receiver6",
				b: components.NewMultiPortReceiverBuilder("receiver6").
					AddPortMapping(components.NewProtocolBuilder("http", 80)).
					AddPortMapping(components.NewProtocolBuilder("grpc", 4317).
						WithTargetPort(4317).
						WithProtocol(corev1.ProtocolTCP).
						WithAppProtocol(&components.GrpcProtocol),
					),
			},
			args: args{
				config: httpAndGrpcConfig,
			},
			want: []corev1.ServicePort{
				{
					Name:        "receiver6-grpc",
					Port:        4317,
					TargetPort:  intstr.FromInt(4317),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: &components.GrpcProtocol,
				},
				{
					Name: "receiver6-http",
					Port: 80,
				},
			},
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "multiple port mappings only one enabled",
			fields: fields{
				name: "receiver6",
				b: components.NewMultiPortReceiverBuilder("receiver6").
					AddPortMapping(components.NewProtocolBuilder("http", 80)).
					AddPortMapping(components.NewProtocolBuilder("grpc", 4317).
						WithTargetPort(4317).
						WithProtocol(corev1.ProtocolTCP).
						WithAppProtocol(&components.GrpcProtocol),
					),
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
			wantBuildErr: assert.NoError,
			wantErr:      assert.NoError,
		},
		{
			name: "error unmarshalling configuration",
			fields: fields{
				name: "receiver1",
				b:    components.NewMultiPortReceiverBuilder("receiver1"),
			},
			args: args{
				config: "invalid config", // Simulate an invalid config that causes LoadMap to fail
			},
			want:         nil,
			wantBuildErr: assert.NoError,
			wantErr:      assert.Error,
		},
		{
			name: "error marshaling configuration",
			fields: fields{
				name: "receiver1",
				b:    components.NewMultiPortReceiverBuilder("receiver1"),
			},
			args: args{
				config: func() {}, // Simulate an invalid config that causes LoadMap to fail
			},
			want:         nil,
			wantBuildErr: assert.NoError,
			wantErr:      assert.Error,
		},
		{
			name: "unknown protocol",
			fields: fields{
				name: "receiver2",
				b:    components.NewMultiPortReceiverBuilder("receiver2").AddPortMapping(components.NewProtocolBuilder("http", 80)),
			},
			args: args{
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"unknown": map[string]interface{}{},
					},
				},
			},
			want:         nil,
			wantBuildErr: assert.NoError,
			wantErr:      assert.Error,
		},
		{
			name: "no name set",
			fields: fields{
				name: "receiver2",
				b:    components.MultiPortBuilder[*components.MultiProtocolEndpointConfig]{},
			},
			args: args{
				config: map[string]interface{}{},
			},
			want:         nil,
			wantBuildErr: assert.Error,
		},
		{
			name: "bad builder",
			fields: fields{
				name: "receiver2",
				b:    components.NewMultiPortReceiverBuilder("receiver2").AddPortMapping(components.NewBuilder[*components.MultiProtocolEndpointConfig]()),
			},
			args: args{
				config: map[string]interface{}{},
			},
			want:         nil,
			wantErr:      assert.NoError,
			wantBuildErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := tt.fields.b.Build()
			if tt.wantBuildErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) && err != nil {
				return
			}
			got, err := s.Ports(logr.Discard(), tt.fields.name, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("Ports(%v)", tt.args.config)) {
				return
			}
			assert.ElementsMatchf(t, tt.want, got, "Ports(%v)", tt.args.config)
			rbacGen, err := s.GetClusterRoleRules(logr.Discard(), tt.args.config)
			assert.NoError(t, err)
			assert.Nil(t, rbacGen)
			livenessProbe, livenessErr := s.GetLivenessProbe(logr.Discard(), tt.args.config)
			assert.NoError(t, livenessErr)
			assert.Nil(t, livenessProbe)
			readinessProbe, readinessErr := s.GetReadinessProbe(logr.Discard(), tt.args.config)
			assert.NoError(t, readinessErr)
			assert.Nil(t, readinessProbe)
		})
	}
}

func TestMultiPortReceiver_GetDefaultConfig(t *testing.T) {
	type args struct {
		logger logr.Logger
		config interface{}
	}
	type testCase struct {
		name    string
		m       components.Parser
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}

	tests := []testCase{
		{
			name: "default config with single protocol settings",
			m: components.NewMultiPortReceiverBuilder("receiver1").
				AddPortMapping(components.NewProtocolBuilder("http", 80).
					WithDefaultRecAddress("0.0.0.0").
					WithTargetPort(8080)).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": nil,
					},
				},
			},
			want: map[string]interface{}{
				"protocols": map[string]interface{}{
					"http": map[string]interface{}{
						"endpoint": "0.0.0.0:80",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "default config with multiple protocol settings",
			m: components.NewMultiPortReceiverBuilder("receiver1").
				AddPortMapping(components.NewProtocolBuilder("http", 80).
					WithDefaultRecAddress("0.0.0.0").
					WithTargetPort(8080)).
				AddPortMapping(components.NewProtocolBuilder("grpc", 90).
					WithDefaultRecAddress("0.0.0.0").
					WithTargetPort(8080)).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": nil,
						"grpc": nil,
					},
				},
			},
			want: map[string]interface{}{
				"protocols": map[string]interface{}{
					"http": map[string]interface{}{
						"endpoint": "0.0.0.0:80",
					},
					"grpc": map[string]interface{}{
						"endpoint": "0.0.0.0:90",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "default config with multiple protocol settings does not override",
			m: components.NewMultiPortReceiverBuilder("receiver1").
				AddPortMapping(components.NewProtocolBuilder("http", 80).
					WithDefaultRecAddress("0.0.0.0").
					WithTargetPort(8080)).
				AddPortMapping(components.NewProtocolBuilder("grpc", 90).
					WithDefaultRecAddress("0.0.0.0").
					WithTargetPort(8080)).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": map[string]interface{}{
							"endpoint": "0.0.0.0:8080",
						},
						"grpc": map[string]interface{}{
							"endpoint": "0.0.0.0:9090",
						},
					},
				},
			},
			want: map[string]interface{}{
				"protocols": map[string]interface{}{
					"http": map[string]interface{}{
						"endpoint": "0.0.0.0:8080",
					},
					"grpc": map[string]interface{}{
						"endpoint": "0.0.0.0:9090",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "config with unknown protocol",
			m:    components.NewMultiPortReceiverBuilder("receiver1").MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"unknown": map[string]interface{}{
							"endpoint": "http://localhost",
						},
					},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "config with missing default service port",
			m:    components.NewMultiPortReceiverBuilder("receiver1").MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"protocols": map[string]interface{}{
						"http": map[string]interface{}{
							"listen_address": "0.0.0.0:8080",
						},
					},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "invalid config fails to decode",
			m:    components.NewMultiPortReceiverBuilder("receiver1").MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: "invalid_config",
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.m.GetDefaultConfig(tt.args.logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetDefaultConfig(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetDefaultConfig(%v, %v)", tt.args.logger, tt.args.config)
		})
	}
}

func TestMultiMustBuildPanics(t *testing.T) {
	b := components.MultiPortBuilder[*components.MultiProtocolEndpointConfig]{}
	assert.Panics(t, func() {
		b.MustBuild()
	})
}
