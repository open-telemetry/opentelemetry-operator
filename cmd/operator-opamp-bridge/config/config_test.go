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

package config

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "base case",
			args: args{
				file: "./testdata/agent.yaml",
			},
			want: &Config{
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				Capabilities: []string{
					"AcceptsRemoteConfig",
					"ReportsEffectiveConfig",
					"ReportsOwnTraces",
					"ReportsOwnMetrics",
					"ReportsOwnLogs",
					"AcceptsOpAMPConnectionSettings",
					"AcceptsOtherConnectionSettings",
					"AcceptsRestartCommand",
					"ReportsHealth",
					"ReportsRemoteConfig",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "http base case",
			args: args{
				file: "./testdata/agenthttpbasic.yaml",
			},
			want: &Config{
				RootLogger: logr.Discard(),
				Endpoint:   "http://127.0.0.1:4320/v1/opamp",
				Capabilities: []string{
					"AcceptsRemoteConfig",
					"ReportsEffectiveConfig",
					"ReportsOwnTraces",
					"ReportsOwnMetrics",
					"ReportsOwnLogs",
					"AcceptsOpAMPConnectionSettings",
					"AcceptsOtherConnectionSettings",
					"AcceptsRestartCommand",
					"ReportsHealth",
					"ReportsRemoteConfig",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "basic components allowed",
			args: args{
				file: "./testdata/agentbasiccomponentsallowed.yaml",
			},
			want: &Config{
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				Capabilities: []string{
					"AcceptsRemoteConfig",
					"ReportsEffectiveConfig",
					"ReportsOwnTraces",
					"ReportsOwnMetrics",
					"ReportsOwnLogs",
					"AcceptsOpAMPConnectionSettings",
					"AcceptsOtherConnectionSettings",
					"AcceptsRestartCommand",
					"ReportsHealth",
					"ReportsRemoteConfig",
				},
				ComponentsAllowed: map[string][]string{
					"receivers": {
						"otlp",
					},
					"processors": {
						"memory_limiter",
						"batch",
					},
					"exporters": {
						"logging",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "bad configuration",
			args: args{
				file: "./testdata/agentbadconf.yaml",
			},
			want: &Config{
				// We do unmarshal partially
				Endpoint: "http://127.0.0.1:4320/v1/opamp",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "error unmarshaling YAML", i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfig(logr.Discard())
			err := LoadFromFile(got, tt.args.file)
			if !tt.wantErr(t, err, fmt.Sprintf("Load(%v)", tt.args.file)) {
				return
			}
			// there are some fields we don't care about, so we ignore them.
			got.ClusterConfig = tt.want.ClusterConfig
			got.RootLogger = tt.want.RootLogger
			assert.Equalf(t, tt.want, got, "Load(%v)", tt.args.file)

		})
	}
}
