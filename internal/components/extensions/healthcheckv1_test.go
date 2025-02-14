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
