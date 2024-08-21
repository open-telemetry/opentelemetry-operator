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
			g:    components.NewGenericParser[*components.SingleEndpointConfig]("test", 0, components.ParseSingleEndpoint),
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
			g:    components.NewGenericParser[*components.SingleEndpointConfig]("test", 0, components.ParseSingleEndpoint),
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
			name: "valid config with listen_address with option",
			g:    components.NewGenericParser[*components.SingleEndpointConfig]("test", 0, components.ParseSingleEndpoint, components.WithProtocol(corev1.ProtocolUDP)),
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
			g:    components.NewGenericParser[*components.SingleEndpointConfig]("test", 0, components.ParseSingleEndpoint),
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
