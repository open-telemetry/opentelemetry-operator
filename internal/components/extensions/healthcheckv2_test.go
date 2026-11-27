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

func TestHealthCheckV2Probe(t *testing.T) {
	type args struct {
		config any
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Probe
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "V2 mode with default status endpoint",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http":   map[string]any{},
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt32(defaultHealthcheckV2Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode with custom status path and port",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http": map[string]any{
						"endpoint": "127.0.0.1:8080",
						"status": map[string]any{
							"path": "/health/status",
						},
					},
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health/status",
						Port: intstr.FromInt32(8080),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode with status disabled and config enabled",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http": map[string]any{
						"status": map[string]any{
							"enabled": false,
						},
						"config": map[string]any{
							"enabled": true,
						},
					},
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/config",
						Port: intstr.FromInt32(defaultHealthcheckV2Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode with status and config disabled",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http": map[string]any{
						"status": map[string]any{
							"enabled": false,
						},
					},
				},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode without http config",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"grpc": map[string]any{
						"endpoint": "localhost:13132",
					},
				},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "Legacy mode with default path and port",
			args: args{
				config: map[string]any{},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(defaultHealthcheckV2Port),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Legacy mode with custom path and port",
			args: args{
				config: map[string]any{
					"endpoint": "127.0.0.1:9090",
					"path":     "/healthz",
				},
			},
			want: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(9090),
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid endpoint",
			args: args{
				config: map[string]any{
					"endpoint": 123,
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("healthcheckv2")
			got, err := parser.GetLivenessProbe(logr.Discard(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetLivenessProbe(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLivenessProbe(%v)", tt.args.config)
		})
	}
}

func TestHealthCheckV2AddressDefaulter(t *testing.T) {
	type args struct {
		config any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "V2 mode with empty http config",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http":   map[string]any{},
				},
			},
			want: map[string]any{
				"use_v2": true,
				"http": map[string]any{
					"endpoint": fmt.Sprintf("%s:%d", components.DefaultRecAddress, defaultHealthcheckV2Port),
					"status": map[string]any{
						"path": defaultHealthcheckV2StatusPath,
					},
					"config": map[string]any{
						"path": defaultHealthcheckV2ConfigPath,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode with custom status path",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http": map[string]any{
						"status": map[string]any{
							"path": "/health/status",
						},
					},
				},
			},
			want: map[string]any{
				"use_v2": true,
				"http": map[string]any{
					"endpoint": fmt.Sprintf("%s:%d", components.DefaultRecAddress, defaultHealthcheckV2Port),
					"status": map[string]any{
						"path": "/health/status",
					},
					"config": map[string]any{
						"path": defaultHealthcheckV2ConfigPath,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode with endpoint port only",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"http": map[string]any{
						"endpoint": ":8080",
					},
				},
			},
			want: map[string]any{
				"use_v2": true,
				"http": map[string]any{
					"endpoint": fmt.Sprintf("%s:8080", components.DefaultRecAddress),
					"status": map[string]any{
						"path": defaultHealthcheckV2StatusPath,
					},
					"config": map[string]any{
						"path": defaultHealthcheckV2ConfigPath,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "V2 mode without http config",
			args: args{
				config: map[string]any{
					"use_v2": true,
					"grpc": map[string]any{
						"endpoint": "localhost:13132",
					},
				},
			},
			want: map[string]any{
				"use_v2": true,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Legacy mode with empty endpoint and path",
			args: args{
				config: map[string]any{},
			},
			want: map[string]any{
				"endpoint": fmt.Sprintf("%s:%d", components.DefaultRecAddress, defaultHealthcheckV2Port),
				"path":     defaultHealthcheckV2Path,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Legacy mode with custom endpoint and path",
			args: args{
				config: map[string]any{
					"endpoint": "127.0.0.1:9090",
					"path":     "/healthz",
				},
			},
			want: map[string]any{
				"endpoint": "127.0.0.1:9090",
				"path":     "/healthz",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid endpoint type",
			args: args{
				config: map[string]any{
					"endpoint": 123,
				},
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("healthcheckv2")
			got, err := parser.GetDefaultConfig(logr.Discard(), tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetDefaultConfig(%v)", tt.args.config)) {
				return
			}

			gotMap, ok := got.(map[string]any)
			if ok {
				assert.Equalf(t, tt.want, gotMap, "GetDefaultConfig(%v)", tt.args.config)
			} else if tt.want != nil {
				t.Errorf("Expected map[string]interface{}, got %T", got)
			}
		})
	}
}
