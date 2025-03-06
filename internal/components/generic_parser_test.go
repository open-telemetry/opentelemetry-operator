// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components_test

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestGenericParser_GetPorts(t *testing.T) {
	type args struct {
		logger logr.Logger
		config interface{}
	}
	type testCase[T any] struct {
		name    string
		g       *components.GenericParser[T]
		args    args
		want    []corev1.ServicePort
		wantErr assert.ErrorAssertionFunc
	}

	tests := []testCase[*components.SingleEndpointConfig]{
		{
			name: "valid config with endpoint",
			g:    components.NewSinglePortParserBuilder("test", 0).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "http://localhost:8080",
				},
			},
			want: []corev1.ServicePort{
				{
					Name: "test",
					Port: 8080,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid config with listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"listen_address": "0.0.0.0:9090",
				},
			},
			want: []corev1.ServicePort{
				{
					Name: "test",
					Port: 9090,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid config with listen_address with settings",
			g:    components.NewSinglePortParserBuilder("test", 0).WithProtocol(corev1.ProtocolUDP).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"listen_address": "0.0.0.0:9090",
				},
			},
			want: []corev1.ServicePort{
				{
					Name:     "test",
					Port:     9090,
					Protocol: corev1.ProtocolUDP,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid config with no endpoint or listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{},
			},
			want:    []corev1.ServicePort{},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.g.Ports(tt.args.logger, "test", tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRBACRules(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetRBACRules(%v, %v)", tt.args.logger, tt.args.config)
		})
	}
}

func TestGenericParser_GetRBACRules(t *testing.T) {
	type args struct {
		logger logr.Logger
		config interface{}
	}
	type testCase[T any] struct {
		name    string
		g       *components.GenericParser[T]
		args    args
		want    []rbacv1.PolicyRule
		wantErr assert.ErrorAssertionFunc
	}

	rbacGenFunc := func(logger logr.Logger, config *components.SingleEndpointConfig) ([]rbacv1.PolicyRule, error) {
		if config.Endpoint == "" && config.ListenAddress == "" {
			return nil, fmt.Errorf("either endpoint or listen_address must be specified")
		}
		return []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		}, nil
	}

	tests := []testCase[*components.SingleEndpointConfig]{
		{
			name: "valid config with endpoint",
			g:    components.NewSinglePortParserBuilder("test", 0).WithClusterRoleRulesGen(rbacGenFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "http://localhost:8080",
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid config with listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).WithClusterRoleRulesGen(rbacGenFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"listen_address": "0.0.0.0:9090",
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid config with no endpoint or listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).WithClusterRoleRulesGen(rbacGenFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "Generic works",
			g:    components.NewBuilder[*components.SingleEndpointConfig]().WithName("test").MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "failed to parse config",
			g:    components.NewSinglePortParserBuilder("test", 0).WithClusterRoleRulesGen(rbacGenFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: func() {},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.g.GetClusterRoleRules(tt.args.logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetClusterRoleRules(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetRBACRules(%v, %v)", tt.args.logger, tt.args.config)
		})
	}
}

func TestGenericParser_GetProbe(t *testing.T) {
	type args struct {
		logger logr.Logger
		config interface{}
	}
	type testCase[T any] struct {
		name             string
		g                *components.GenericParser[T]
		args             args
		livenessProbe    *corev1.Probe
		readinessProbe   *corev1.Probe
		wantLivenessErr  assert.ErrorAssertionFunc
		wantReadinessErr assert.ErrorAssertionFunc
	}
	probeFunc := func(logger logr.Logger, config *components.SingleEndpointConfig) (*corev1.Probe, error) {
		if config.Endpoint == "" && config.ListenAddress == "" {
			return nil, fmt.Errorf("either endpoint or listen_address must be specified")
		}
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/hello",
					Port: intstr.FromInt32(8080),
				},
			},
		}, nil
	}

	tests := []testCase[*components.SingleEndpointConfig]{
		{
			name: "valid config with endpoint",
			g:    components.NewSinglePortParserBuilder("test", 0).WithReadinessGen(probeFunc).WithLivenessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "http://localhost:8080",
				},
			},
			livenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/hello",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			readinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/hello",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			wantLivenessErr:  assert.NoError,
			wantReadinessErr: assert.NoError,
		},
		{
			name: "valid config with listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).WithReadinessGen(probeFunc).WithLivenessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"listen_address": "0.0.0.0:9090",
				},
			},
			livenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/hello",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			readinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/hello",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			wantLivenessErr:  assert.NoError,
			wantReadinessErr: assert.NoError,
		},
		{
			name: "readiness invalid config with no endpoint or listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).WithReadinessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{},
			},
			readinessProbe:   nil,
			livenessProbe:    nil,
			wantReadinessErr: assert.Error,
			wantLivenessErr:  assert.NoError,
		},
		{
			name: "liveness invalid config with no endpoint or listen_address",
			g:    components.NewSinglePortParserBuilder("test", 0).WithLivenessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{},
			},
			readinessProbe:   nil,
			livenessProbe:    nil,
			wantReadinessErr: assert.NoError,
			wantLivenessErr:  assert.Error,
		},
		{
			name: "liveness failed to parse config",
			g:    components.NewSinglePortParserBuilder("test", 0).WithLivenessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: func() {},
			},
			livenessProbe:    nil,
			readinessProbe:   nil,
			wantLivenessErr:  assert.Error,
			wantReadinessErr: assert.NoError,
		},
		{
			name: "readiness failed to parse config",
			g:    components.NewSinglePortParserBuilder("test", 0).WithReadinessGen(probeFunc).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: func() {},
			},
			livenessProbe:    nil,
			readinessProbe:   nil,
			wantLivenessErr:  assert.NoError,
			wantReadinessErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			livenessProbe, err := tt.g.GetLivenessProbe(tt.args.logger, tt.args.config)
			if !tt.wantLivenessErr(t, err, fmt.Sprintf("GetLivenessProbe(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.livenessProbe, livenessProbe, "GetLivenessProbe(%v, %v)", tt.args.logger, tt.args.config)
			readinessProbe, err := tt.g.GetReadinessProbe(tt.args.logger, tt.args.config)
			if !tt.wantReadinessErr(t, err, fmt.Sprintf("GetReadinessProbe(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.readinessProbe, readinessProbe, "GetReadinessProbe(%v, %v)", tt.args.logger, tt.args.config)
		})
	}
}

func TestGenericParser_GetDefaultConfig(t *testing.T) {
	type args struct {
		logger logr.Logger
		config interface{}
	}
	type testCase[T any] struct {
		name    string
		g       *components.GenericParser[T]
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}

	tests := []testCase[*components.SingleEndpointConfig]{
		{
			name: "no settings or defaultsApplier returns config",
			g:    &components.GenericParser[*components.SingleEndpointConfig]{},
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "http://localhost:8080",
				},
			},
			want: map[string]interface{}{
				"endpoint": "http://localhost:8080",
			},
			wantErr: assert.NoError,
		},
		{
			name: "empty defaultRecAddr returns config",
			g:    components.NewSinglePortParserBuilder("test", 0).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "http://localhost:8080",
				},
			},
			want: map[string]interface{}{
				"endpoint": "http://localhost:8080",
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid settings with defaultsApplier",
			g:    components.NewSinglePortParserBuilder("test", 8080).WithDefaultRecAddress("127.0.0.1").WithDefaultsApplier(components.AddressDefaulter).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": nil,
				},
			},
			want: map[string]interface{}{
				"endpoint": "127.0.0.1:8080",
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid settings with defaultsApplier doesnt override",
			g:    components.NewSinglePortParserBuilder("test", 8080).WithDefaultRecAddress("127.0.0.1").WithDefaultsApplier(components.AddressDefaulter).MustBuild(),
			args: args{
				logger: logr.Discard(),
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:9090",
				},
			},
			want: map[string]interface{}{
				"endpoint": "127.0.0.1:9090",
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid config fails to decode",
			g:    components.NewSinglePortParserBuilder("test", 8080).WithDefaultRecAddress("127.0.0.1").WithDefaultsApplier(components.AddressDefaulter).MustBuild(),
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
			got, err := tt.g.GetDefaultConfig(tt.args.logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetDefaultConfig(%v, %v)", tt.args.logger, tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetDefaultConfig(%v, %v)", tt.args.logger, tt.args.config)
		})
	}
}
