// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadPriority(t *testing.T) {
	t.Run("default values when nothing is set except config yaml", func(t *testing.T) {
		args := []string{
			"--" + configFilePathFlagName + "=./testdata/agent.yaml",
			"--" + kubeConfigPathFlagName + "=./testdata/kubeconfig.yaml",
		}
		cfg, err := Load(GetLogger(), args)

		assert.NoError(t, err)
		assert.Equal(t, defaultServerListenAddr, cfg.ListenAddr, "use default value")
		assert.Equal(t, defaultHealthListenAddr, cfg.HealthListenAddr, "use default value")
		assert.Equal(t, defaultHeartbeatInterval, cfg.HeartbeatInterval, "use default value")
		assert.Equal(t, opampBridgeName, cfg.Name, "use default value")
	})
	t.Run("command-line has priority over config file for time.Duration values", func(t *testing.T) {
		args := []string{
			"--" + configFilePathFlagName + "=./testdata/agenthttpbasic.yaml",
			"--" + kubeConfigPathFlagName + "=./testdata/kubeconfig.yaml",
			"--" + heartbeatIntervalFlagName + "=10s",
		}
		cfg, err := Load(GetLogger(), args)

		assert.NoError(t, err)
		assert.Equal(t, defaultServerListenAddr, cfg.ListenAddr, "use default value")
		assert.Equal(t, defaultHealthListenAddr, cfg.HealthListenAddr, "use default value")
		assert.Equal(t, 10*time.Second, cfg.HeartbeatInterval, "command-line priority is higher than config, overwrite time.Duration value")
		assert.Equal(t, "http-test-bridge", cfg.Name, "config file priority is higher than default string value")
	})

	t.Run("command-line has priority over config file for string values", func(t *testing.T) {
		testOpAMPBridgeName := "opamp-bridge-name-test"
		args := []string{
			"--" + configFilePathFlagName + "=./testdata/agenthttpbasic.yaml",
			"--" + kubeConfigPathFlagName + "=./testdata/kubeconfig.yaml",
			"--" + nameFlagName + "=" + testOpAMPBridgeName,
			"--" + healthListenAddrFlagName + "=:8082",
		}
		cfg, err := Load(GetLogger(), args)

		assert.NoError(t, err)
		assert.Equal(t, defaultServerListenAddr, cfg.ListenAddr, "use default value")
		assert.Equal(t, ":8082", cfg.HealthListenAddr, "command-line priority is higher than config, overwrite health address")
		assert.Equal(t, 45*time.Second, cfg.HeartbeatInterval, "config file priority is higher than default time.Duration value")
		assert.Equal(t, testOpAMPBridgeName, cfg.Name, "command-line priority is higher than config, overwrite string value")
	})
}

func TestLoadFromFileHealthListenAddr(t *testing.T) {
	cfg := NewConfig(logr.Discard())
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(`
endpoint: ws://127.0.0.1:4320/v1/opamp
healthListenAddr: ":9090"
capabilities:
  ReportsHealth: true
`), 0o600))

	require.NoError(t, LoadFromFile(cfg, configFile))

	assert.Equal(t, ":9090", cfg.HealthListenAddr)
	assert.Equal(t, defaultServerListenAddr, cfg.ListenAddr)
}

func TestLoadFromFile(t *testing.T) {
	type args struct {
		file         string
		envVariables map[string]string
	}
	instanceId := uuid.New()
	tests := []struct {
		name    string
		args    args
		want    *Config
		needErr bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "base case",
			args: args{
				file: "./testdata/agent.yaml",
			},
			want: &Config{
				instanceId:         instanceId,
				RootLogger:         logr.Discard(),
				ListenAddr:         defaultServerListenAddr,
				HealthListenAddr:   defaultHealthListenAddr,
				KubeConfigFilePath: defaultKubeConfigPath,
				Endpoint:           "ws://127.0.0.1:4320/v1/opamp",
				HeartbeatInterval:  defaultHeartbeatInterval,
				Name:               opampBridgeName,
				Mode:               defaultMode,
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
			needErr: false,
			wantErr: assert.NoError,
		},
		{
			name: "http base case",
			args: args{
				file: "./testdata/agenthttpbasic.yaml",
			},
			want: &Config{
				instanceId:         instanceId,
				RootLogger:         logr.Discard(),
				Endpoint:           "http://127.0.0.1:4320/v1/opamp",
				ListenAddr:         defaultServerListenAddr,
				HealthListenAddr:   defaultHealthListenAddr,
				KubeConfigFilePath: defaultKubeConfigPath,
				HeartbeatInterval:  45 * time.Second,
				Name:               "http-test-bridge",
				Mode:               defaultMode,
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
			needErr: false,
			wantErr: assert.NoError,
		},
		{
			name: "basic components allowed",
			args: args{
				file: "./testdata/agentbasiccomponentsallowed.yaml",
			},
			want: &Config{
				instanceId:         instanceId,
				RootLogger:         logr.Discard(),
				Endpoint:           "ws://127.0.0.1:4320/v1/opamp",
				ListenAddr:         defaultServerListenAddr,
				HealthListenAddr:   defaultHealthListenAddr,
				KubeConfigFilePath: defaultKubeConfigPath,
				HeartbeatInterval:  defaultHeartbeatInterval,
				Name:               opampBridgeName,
				Mode:               defaultMode,
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
			needErr: false,
			wantErr: assert.NoError,
		},
		{
			name: "bad configuration",
			args: args{
				file: "./testdata/agentbadconf.yaml",
			},
			want:    &Config{},
			needErr: true,
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				if err == nil {
					return assert.Fail(t, "expected YAML error, got nil", i...)
				}
				msg := err.Error()
				if !strings.Contains(msg, "error unmarshaling YAML") {
					return assert.Fail(t, fmt.Sprintf("unexpected error %q", msg), i...)
				}
				return true
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
				ListenAddr:         defaultServerListenAddr,
				HealthListenAddr:   defaultHealthListenAddr,
				KubeConfigFilePath: defaultKubeConfigPath,
				HeartbeatInterval:  defaultHeartbeatInterval,
				Name:               opampBridgeName,
				Mode:               defaultMode,
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
			needErr: false,
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
				ListenAddr:         defaultServerListenAddr,
				HealthListenAddr:   defaultHealthListenAddr,
				KubeConfigFilePath: defaultKubeConfigPath,
				HeartbeatInterval:  defaultHeartbeatInterval,
				Name:               opampBridgeName,
				Mode:               defaultMode,
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
			needErr: false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envVariables != nil {
				for key, value := range tt.args.envVariables {
					t.Setenv(key, value)
				}
			}
			got := NewConfig(logr.Discard())
			err := LoadFromFile(got, tt.args.file)
			if tt.needErr {
				_ = tt.wantErr(t, err, fmt.Sprintf("Load(%v)", tt.args.file))
				return
			}
			got.instanceId = tt.want.instanceId
			got.ClusterConfig = tt.want.ClusterConfig
			got.RootLogger = tt.want.RootLogger
			assert.Equalf(t, tt.want, got, "Load(%v)", tt.args.file)
		})
	}
}

func TestLoadFromFileRejectsDuplicateStandaloneConfigKeys(t *testing.T) {
	cfg := []byte(`
mode: standalone
standalone:
  agents:
    - namespace: default
      type: otel-collector
      workloadRef:
        apiVersion: apps/v1
        kind: Deployment
        name: collector
      config:
        collector:
          kind: configmap
          name: collector-config
          key: collector.yaml
        collector:
          kind: configmap
          name: other-config
          key: other.yaml
`)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, cfg, 0o600))

	err := LoadFromFile(NewConfig(logr.Discard()), configPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, `mapping key "collector" already defined`)
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

func TestNewConfigSetsDefaultMode(t *testing.T) {
	cfg := NewConfig(logr.Discard())
	assert.Equal(t, operatorMode, cfg.Mode, "NewConfig should seed Mode with the documented default so logs/state match the flag default")
	assert.False(t, cfg.IsStandaloneMode())
}

func TestValidateRejectsUnknownMode(t *testing.T) {
	t.Run("typo on mode is rejected", func(t *testing.T) {
		cfg := NewConfig(logr.Discard())
		cfg.Mode = "standlon"
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, `invalid mode "standlon"`)
	})

	t.Run("case-sensitive mode is rejected", func(t *testing.T) {
		cfg := NewConfig(logr.Discard())
		cfg.Mode = "Standalone"
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, `invalid mode "Standalone"`)
	})

	t.Run("operator mode validates without error", func(t *testing.T) {
		cfg := NewConfig(logr.Discard())
		cfg.Mode = operatorMode
		assert.NoError(t, cfg.Validate())
	})

	t.Run("empty mode is normalized to the documented default", func(t *testing.T) {
		cfg := NewConfig(logr.Discard())
		cfg.Mode = ""
		require.NoError(t, cfg.Validate())
		assert.Equal(t, operatorMode, cfg.Mode)
	})

	t.Run("invalid mode does not fall through to operator behavior", func(t *testing.T) {
		cfg := NewConfig(logr.Discard())
		cfg.Mode = "standlon"
		cfg.Standalone = StandaloneConfig{Agents: []StandaloneAgentConfig{}}
		// Previously, an unknown mode would short-circuit Validate() at the
		// `IsStandaloneMode()` guard, silently behaving as operator. Make sure
		// the error is surfaced instead.
		err := cfg.Validate()
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "standalone mode requires at least one configured agent")
	})
}

func TestNewStandaloneAgentConfigUsesWorkloadRefNameAsHostName(t *testing.T) {
	cfg := NewConfig(logr.Discard())
	cfg.Mode = standaloneMode
	cfg.Headers = Headers{"x-test-header": "header-value"}
	cfg.Capabilities = map[Capability]bool{AcceptsRemoteConfig: true}
	cfg.ComponentsAllowed = map[string][]string{"receivers": {"otlp"}}
	cfg.AgentDescription.NonIdentifyingAttributes = map[string]string{"environment": "test"}

	agentCfg := NewStandaloneAgentConfig(cfg, StandaloneAgentConfig{
		Namespace: "default",
		Type:      "otel-collector",
		WorkloadRef: StandaloneWorkloadRef{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "collector-workload",
		},
	})

	desc := agentCfg.GetDescription()
	assert.Contains(t, desc.NonIdentifyingAttributes, &protobufs.KeyValue{Key: "host.name", Value: &protobufs.AnyValue{
		Value: &protobufs.AnyValue_StringValue{StringValue: "collector-workload"},
	}})
	assert.NotContains(t, desc.NonIdentifyingAttributes, &protobufs.KeyValue{Key: "host.name", Value: &protobufs.AnyValue{
		Value: &protobufs.AnyValue_StringValue{StringValue: hostname},
	}})

	agentCfg.Headers["x-test-header"] = "changed"
	agentCfg.Capabilities[AcceptsRemoteConfig] = false
	agentCfg.ComponentsAllowed["receivers"][0] = "prometheus"
	agentCfg.AgentDescription.NonIdentifyingAttributes["environment"] = "changed"

	assert.Equal(t, "header-value", cfg.Headers["x-test-header"])
	assert.True(t, cfg.Capabilities[AcceptsRemoteConfig])
	assert.Equal(t, []string{"otlp"}, cfg.ComponentsAllowed["receivers"])
	assert.Equal(t, "test", cfg.AgentDescription.NonIdentifyingAttributes["environment"])
}
