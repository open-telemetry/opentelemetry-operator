// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	type args struct {
		file         string
		envVariables map[string]string
	}
	instanceId := uuid.New()
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
				instanceId: instanceId,
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				Capabilities: map[Capability]bool{
					AcceptsRemoteConfig:            true,
					ReportsEffectiveConfig:         true,
					ReportsOwnTraces:               true,
					ReportsOwnMetrics:              true,
					ReportsOwnLogs:                 true,
					AcceptsOpAMPConnectionSettings: true,
					AcceptsOtherConnectionSettings: true,
					AcceptsRestartCommand:          true,
					ReportsHealth:                  true,
					ReportsRemoteConfig:            true,
					AcceptsPackages:                false,
					ReportsPackageStatuses:         false,
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
				instanceId:        instanceId,
				RootLogger:        logr.Discard(),
				Endpoint:          "http://127.0.0.1:4320/v1/opamp",
				HeartbeatInterval: 45 * time.Second,
				Name:              "http-test-bridge",
				Capabilities: map[Capability]bool{
					AcceptsRemoteConfig:            true,
					ReportsEffectiveConfig:         true,
					ReportsOwnTraces:               true,
					ReportsOwnMetrics:              true,
					ReportsOwnLogs:                 true,
					AcceptsOpAMPConnectionSettings: true,
					AcceptsOtherConnectionSettings: true,
					AcceptsRestartCommand:          true,
					ReportsHealth:                  true,
					ReportsRemoteConfig:            true,
					AcceptsPackages:                false,
					ReportsPackageStatuses:         false,
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
				instanceId: instanceId,
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				Capabilities: map[Capability]bool{
					AcceptsRemoteConfig:            true,
					ReportsEffectiveConfig:         true,
					ReportsOwnTraces:               true,
					ReportsOwnMetrics:              true,
					ReportsOwnLogs:                 true,
					AcceptsOpAMPConnectionSettings: true,
					AcceptsOtherConnectionSettings: true,
					AcceptsRestartCommand:          true,
					ReportsHealth:                  true,
					ReportsRemoteConfig:            true,
					AcceptsPackages:                false,
					ReportsPackageStatuses:         false,
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
						"debug",
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
			want: &Config{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "error unmarshaling YAML", i...)
			},
		},
		{
			name: "base case with headers",
			args: args{
				file: "./testdata/agentwithheaders.yaml",
				envVariables: map[string]string{
					"MY_ENV_VAR_1": "my-env-variable-1-value",
					"MY_ENV_VAR_2": "my-env-variable-2-value",
				},
			},
			want: &Config{
				instanceId: instanceId,
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				Headers: map[string]string{
					"authentication":    "access-12345-token",
					"my-header-key":     "my-header-value",
					"my-env-variable-1": "my-env-variable-1-value",
					"my-env-variable-2": "my-env-variable-2-value",
				},
				Capabilities: map[Capability]bool{
					AcceptsRemoteConfig:            true,
					ReportsEffectiveConfig:         true,
					ReportsOwnTraces:               true,
					ReportsOwnMetrics:              true,
					ReportsOwnLogs:                 true,
					AcceptsOpAMPConnectionSettings: true,
					AcceptsOtherConnectionSettings: true,
					AcceptsRestartCommand:          true,
					ReportsHealth:                  true,
					ReportsRemoteConfig:            true,
					AcceptsPackages:                false,
					ReportsPackageStatuses:         false,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "base case with nonidentify attributes",
			args: args{
				file: "./testdata/agentwithdescription.yaml",
				envVariables: map[string]string{
					"MY_ENV_VAR_1": "my-env-variable-1-value",
					"MY_ENV_VAR_2": "my-env-variable-2-value",
				},
			},
			want: &Config{
				instanceId: instanceId,
				RootLogger: logr.Discard(),
				Endpoint:   "ws://127.0.0.1:4320/v1/opamp",
				AgentDescription: AgentDescription{
					NonIdentifyingAttributes: map[string]string{
						"custom.attribute": "custom-value",
					},
				},
				Capabilities: map[Capability]bool{
					AcceptsRemoteConfig:            true,
					ReportsEffectiveConfig:         true,
					ReportsOwnTraces:               true,
					ReportsOwnMetrics:              true,
					ReportsOwnLogs:                 true,
					AcceptsOpAMPConnectionSettings: true,
					AcceptsOtherConnectionSettings: true,
					AcceptsRestartCommand:          true,
					ReportsHealth:                  true,
					ReportsRemoteConfig:            true,
					AcceptsPackages:                false,
					ReportsPackageStatuses:         false,
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envVariables != nil {
				for key, value := range tt.args.envVariables {
					err := os.Setenv(key, value)
					assert.NoError(t, err)
				}
			}
			got := NewConfig(logr.Discard())
			err := LoadFromFile(got, tt.args.file)
			if !tt.wantErr(t, err, fmt.Sprintf("Load(%v)", tt.args.file)) {
				return
			}
			// there are some fields we don't care about, so we ignore them.
			got.instanceId = tt.want.instanceId
			got.ClusterConfig = tt.want.ClusterConfig
			got.RootLogger = tt.want.RootLogger
			assert.Equalf(t, tt.want, got, "Load(%v)", tt.args.file)
		})
	}
}

func TestGetDescription(t *testing.T) {
	got := NewConfig(logr.Discard())
	instanceId := uuid.New()
	got.instanceId = instanceId
	err := LoadFromFile(got, "./testdata/agentwithdescription.yaml")
	require.NoError(t, err, fmt.Sprintf("Load(%v)", "./testdata/agentwithdescription.yaml"))
	desc := got.GetDescription()
	assert.Len(t, desc.IdentifyingAttributes, 3)
	assert.Contains(t, desc.IdentifyingAttributes, &protobufs.KeyValue{Key: "service.instance.id", Value: &protobufs.AnyValue{
		Value: &protobufs.AnyValue_StringValue{StringValue: instanceId.String()},
	}})
	assert.Len(t, desc.NonIdentifyingAttributes, 3)
	assert.Contains(t, desc.NonIdentifyingAttributes, &protobufs.KeyValue{Key: "custom.attribute", Value: &protobufs.AnyValue{
		Value: &protobufs.AnyValue_StringValue{StringValue: "custom-value"},
	}})
}

func TestGetDescriptionNoneSet(t *testing.T) {
	got := NewConfig(logr.Discard())
	instanceId := uuid.New()
	got.instanceId = instanceId
	err := LoadFromFile(got, "./testdata/agent.yaml")
	require.NoError(t, err, fmt.Sprintf("Load(%v)", "./testdata/agent.yaml"))
	desc := got.GetDescription()
	assert.Len(t, desc.IdentifyingAttributes, 3)
	assert.Contains(t, desc.IdentifyingAttributes, &protobufs.KeyValue{Key: "service.instance.id", Value: &protobufs.AnyValue{
		Value: &protobufs.AnyValue_StringValue{StringValue: instanceId.String()},
	}})
	assert.Len(t, desc.NonIdentifyingAttributes, 2)
}
