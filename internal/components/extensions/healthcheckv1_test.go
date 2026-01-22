// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestHealthCheckV1Probe(t *testing.T) {
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Probe
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Valid path and custom port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:8080",
					"path":     "/healthz",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Valid path and default port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1",
					"path":     "/healthz",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(defaultHealthcheckV1Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Empty path and custom port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:9090",
					"path":     "",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(9090),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Empty path and default port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1",
					"path":     "",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(defaultHealthcheckV1Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Nil path and custom port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:7070",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(7070),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Nil path and default port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(defaultHealthcheckV1Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid endpoint",
			args: args{
				config: map[string]interface{}{
					"endpoint": 123,
					"path":     "/healthz",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "Zero custom port, default port fallback",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:0",
					"path":     "/healthz",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(defaultHealthcheckV1Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("health_check")
			got, err := parser.GetLivenessProbe(logr.Discard(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetLivenessProbe(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLivenessProbe(%v)", tt.args.config)
		})
	}
}

func TestHealthCheckV1AddressDefaulter(t *testing.T) {
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Empty endpoint and path",
			args: args{
				config: map[string]interface{}{},
			},
			want: map[string]interface{}{
				"endpoint": fmt.Sprintf("%s:%d", components.DefaultRecAddress, defaultHealthcheckV1Port),
				"path":     defaultHealthcheckV1Path,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Empty endpoint with custom path",
			args: args{
				config: map[string]interface{}{
					"path": "/custom-health",
				},
			},
			want: map[string]interface{}{
				"endpoint": fmt.Sprintf("%s:%d", components.DefaultRecAddress, defaultHealthcheckV1Port),
				"path":     "/custom-health",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Endpoint with port only",
			args: args{
				config: map[string]interface{}{
					"endpoint": ":8080",
				},
			},
			want: map[string]interface{}{
				"endpoint": fmt.Sprintf("%s:8080", components.DefaultRecAddress),
				"path":     defaultHealthcheckV1Path,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Endpoint with custom address and port",
			args: args{
				config: map[string]interface{}{
					"endpoint": "127.0.0.1:9090",
					"path":     "/healthz",
				},
			},
			want: map[string]interface{}{
				"endpoint": "127.0.0.1:9090",
				"path":     "/healthz",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Endpoint with empty address",
			args: args{
				config: map[string]interface{}{
					"endpoint": ":7070",
				},
			},
			want: map[string]interface{}{
				"endpoint": fmt.Sprintf("%s:7070", components.DefaultRecAddress),
				"path":     defaultHealthcheckV1Path,
			},
			wantErr: assert.NoError,
		},
		{
			name: "IPv6 address",
			args: args{
				config: map[string]interface{}{
					"endpoint": "[::1]:8080",
				},
			},
			want: map[string]interface{}{
				"endpoint": "[::1]:8080",
				"path":     defaultHealthcheckV1Path,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid endpoint type",
			args: args{
				config: map[string]interface{}{
					"endpoint": 123,
				},
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("health_check")
			got, err := parser.GetDefaultConfig(logr.Discard(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetDefaultConfig(%v)", tt.args.config)) {
				return
			}

			gotMap, ok := got.(map[string]interface{})
			if ok {
				assert.Equalf(t, tt.want, gotMap, "GetDefaultConfig(%v)", tt.args.config)
			} else if tt.want != nil {
				t.Errorf("Expected map[string]interface{}, got %T", got)
			}
		})
	}
}
