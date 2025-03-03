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

func TestBuilder_Build(t *testing.T) {
	type sampleConfig struct {
		example string
		number  int
		m       map[string]interface{}
	}
	type want struct {
		name           string
		ports          []corev1.ServicePort
		rules          []rbacv1.PolicyRule
		livenessProbe  *corev1.Probe
		readinessProbe *corev1.Probe
	}
	type fields[T any] struct {
		b components.Builder[T]
	}
	type params struct {
		conf interface{}
	}
	type testCase[T any] struct {
		name            string
		fields          fields[T]
		params          params
		want            want
		wantErr         assert.ErrorAssertionFunc
		wantRbacErr     assert.ErrorAssertionFunc
		wantLivenessErr assert.ErrorAssertionFunc
	}
	examplePortParser := func(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config sampleConfig) ([]corev1.ServicePort, error) {
		if defaultPort != nil {
			return []corev1.ServicePort{*defaultPort}, nil
		}
		return nil, nil
	}
	exampleProbeGen := func(logger logr.Logger, config sampleConfig) (*corev1.Probe, error) {
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/hello",
					Port: intstr.FromInt32(8080),
				},
			},
		}, nil
	}
	tests := []testCase[sampleConfig]{
		{
			name: "basic valid configuration",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithPortParser(examplePortParser).
					WithName("test-service").
					WithPort(80).
					WithNodePort(80).
					WithProtocol(corev1.ProtocolTCP),
			},
			params: params{
				conf: sampleConfig{},
			},
			want: want{
				name: "__test-service",
				ports: []corev1.ServicePort{
					{
						Name:     "test-service",
						Port:     80,
						NodePort: 80,
						Protocol: corev1.ProtocolTCP,
					},
				},
				rules: nil,
			},
			wantErr:         assert.NoError,
			wantRbacErr:     assert.NoError,
			wantLivenessErr: assert.NoError,
		},
		{
			name: "missing name",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithPort(8080).
					WithProtocol(corev1.ProtocolUDP),
			},
			params: params{
				conf: sampleConfig{},
			},
			want:            want{},
			wantErr:         assert.Error,
			wantRbacErr:     assert.NoError,
			wantLivenessErr: assert.NoError,
		},
		{
			name: "complete configuration with RBAC rules",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithName("secure-service").
					WithPort(443).
					WithProtocol(corev1.ProtocolTCP).
					WithClusterRoleRulesGen(func(logger logr.Logger, config sampleConfig) ([]rbacv1.PolicyRule, error) {
						rules := []rbacv1.PolicyRule{
							{
								NonResourceURLs: []string{config.example},
								APIGroups:       []string{""},
								Resources:       []string{"pods"},
								Verbs:           []string{"get", "list"},
							},
						}
						if config.number > 100 {
							rules = append(rules, rbacv1.PolicyRule{
								APIGroups: []string{""},
								Resources: []string{"nodes"},
								Verbs:     []string{"get", "list"},
							})
						}
						return rules, nil
					}),
			},
			params: params{
				conf: sampleConfig{
					example: "test",
					number:  100,
					m: map[string]interface{}{
						"key": "value",
					},
				},
			},
			want: want{
				name:  "__secure-service",
				ports: nil,
				rules: []rbacv1.PolicyRule{
					{
						NonResourceURLs: []string{"test"},
						APIGroups:       []string{""},
						Resources:       []string{"pods"},
						Verbs:           []string{"get", "list"},
					},
				},
			},
			wantErr:         assert.NoError,
			wantRbacErr:     assert.NoError,
			wantLivenessErr: assert.NoError,
		},
		{
			name: "complete configuration with RBAC rules errors",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithName("secure-service").
					WithPort(443).
					WithProtocol(corev1.ProtocolTCP).
					WithClusterRoleRulesGen(func(logger logr.Logger, config sampleConfig) ([]rbacv1.PolicyRule, error) {
						rules := []rbacv1.PolicyRule{
							{
								NonResourceURLs: []string{config.example},
								APIGroups:       []string{""},
								Resources:       []string{"pods"},
								Verbs:           []string{"get", "list"},
							},
						}
						if v, ok := config.m["key"]; ok && v == "value" {
							return nil, fmt.Errorf("errors from function")
						}
						return rules, nil
					}),
			},
			params: params{
				conf: sampleConfig{
					example: "test",
					number:  100,
					m: map[string]interface{}{
						"key": "value",
					},
				},
			},
			want: want{
				name:  "__secure-service",
				ports: nil,
				rules: nil,
			},
			wantErr:         assert.NoError,
			wantRbacErr:     assert.Error,
			wantLivenessErr: assert.NoError,
		},
		{
			name: "complete configuration with probe gen",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithName("secure-service").
					WithPort(443).
					WithProtocol(corev1.ProtocolTCP).
					WithLivenessGen(exampleProbeGen).
					WithReadinessGen(exampleProbeGen),
			},
			params: params{
				conf: sampleConfig{
					example: "test",
					number:  100,
					m: map[string]interface{}{
						"key": "value",
					},
				},
			},
			want: want{
				name:  "__secure-service",
				ports: nil,
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
			},
			wantErr:         assert.NoError,
			wantRbacErr:     assert.NoError,
			wantLivenessErr: assert.NoError,
		},
		{
			name: "complete configuration with probe gen errors",
			fields: fields[sampleConfig]{
				b: components.NewBuilder[sampleConfig]().
					WithName("secure-service").
					WithPort(443).
					WithProtocol(corev1.ProtocolTCP).
					WithLivenessGen(func(logger logr.Logger, config sampleConfig) (*corev1.Probe, error) {
						return nil, fmt.Errorf("no probe")
					}),
			},
			params: params{
				conf: sampleConfig{
					example: "test",
					number:  100,
					m: map[string]interface{}{
						"key": "value",
					},
				},
			},
			want: want{
				name:  "__secure-service",
				ports: nil,
				rules: nil,
			},
			wantErr:         assert.NoError,
			wantRbacErr:     assert.NoError,
			wantLivenessErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.b.Build()
			if tt.wantErr(t, err, "WantErr()") && err != nil {
				return
			}
			assert.Equalf(t, tt.want.name, got.ParserName(), "ParserName()")
			ports, err := got.Ports(logr.Discard(), got.ParserType(), tt.params.conf)
			assert.NoError(t, err)
			assert.Equalf(t, tt.want.ports, ports, "Ports()")
			rules, rbacErr := got.GetClusterRoleRules(logr.Discard(), tt.params.conf)
			if tt.wantRbacErr(t, rbacErr, "WantRbacErr()") && rbacErr != nil {
				return
			}
			assert.Equalf(t, tt.want.rules, rules, "GetRBACRules()")
			livenessProbe, livenessErr := got.GetLivenessProbe(logr.Discard(), tt.params.conf)
			if tt.wantLivenessErr(t, livenessErr, "wantLivenessErr()") && livenessErr != nil {
				return
			}
			assert.Equalf(t, tt.want.livenessProbe, livenessProbe, "GetLivenessProbe()")
			readinessProbe, readinessErr := got.GetReadinessProbe(logr.Discard(), tt.params.conf)
			assert.NoError(t, readinessErr)
			assert.Equalf(t, tt.want.readinessProbe, readinessProbe, "GetReadinessProbe()")
		})
	}
}

func TestMustBuildPanics(t *testing.T) {
	b := components.Builder[*components.SingleEndpointConfig]{}
	assert.Panics(t, func() {
		b.MustBuild()
	})
}
