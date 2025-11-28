// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}

	agent := NewAgent(logger, instanceId, conn, nil)
	require.NotNil(t, agent, "agent should not be nil")
	assert.Equal(t, instanceId, agent.InstanceId, "instance ID should match")
	assert.Equal(t, conn, agent.conn, "connection should match")
}

func TestAgent_hasCapability(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn, nil)

	agent.Status = &protobufs.AgentToServer{
		Capabilities: uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig),
	}

	assert.True(t, agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig), "should have capability")
	assert.False(t, agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth), "should not have capability")
}

func TestAgent_UpdateStatus(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn, nil)

	newStatus := &protobufs.AgentToServer{
		SequenceNum: 1,
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: "service.name", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: "test-service"}}},
			},
		},
		//nolint:gosec
		Capabilities: uint64(protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig),
	}

	response := &protobufs.ServerToAgent{}
	agent.UpdateStatus(newStatus, response)

	assert.Equal(t, newStatus, agent.Status, "status should be updated")
	assert.True(t, response.Flags&uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState) != 0, "should request full state")
}

func TestAgent_GetHealth(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn, nil)

	health := &protobufs.ComponentHealth{
		Healthy: true,
	}
	agent.health = health

	assert.Equal(t, health, agent.GetHealth(), "health should match")
}

func TestAgent_GetConfiguration(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn, nil)

	config := &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"config.yaml": {Body: []byte("receivers:\n  otlp:\n")},
			},
		},
	}
	agent.effectiveConfig = config

	assert.Equal(t, config, agent.GetConfiguration(), "configuration should match")
}

func Test_CalcConnectionSettings(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr bool
		expected    *protobufs.ServerToAgent
		setSettings *mockBridgeAgent
	}{
		{
			name: "all settings present",
			expected: &protobufs.ServerToAgent{
				ConnectionSettings: &protobufs.ConnectionSettingsOffers{
					OwnMetrics: &protobufs.TelemetryConnectionSettings{
						DestinationEndpoint: "metrics-endpoint",
					},
					OwnTraces: &protobufs.TelemetryConnectionSettings{
						DestinationEndpoint: "traces-endpoint",
					},
					OwnLogs: &protobufs.TelemetryConnectionSettings{
						DestinationEndpoint: "logs-endpoint",
					},
					OtherConnections: map[string]*protobufs.OtherConnectionSettings{
						"custom-connection": {
							DestinationEndpoint: "custom-endpoint",
							OtherSettings: map[string]string{
								"setting1": "value1",
								"setting2": "value2",
							},
						},
					},
				},
			},
			setSettings: &mockBridgeAgent{
				ownMetrics: &protobufs.TelemetryConnectionSettings{
					DestinationEndpoint: "metrics-endpoint",
				},
				ownTraces: &protobufs.TelemetryConnectionSettings{
					DestinationEndpoint: "traces-endpoint",
				},
				ownLogs: &protobufs.TelemetryConnectionSettings{
					DestinationEndpoint: "logs-endpoint",
				},
				otherConnections: map[string]*protobufs.OtherConnectionSettings{
					"custom-connection": {
						DestinationEndpoint: "custom-endpoint",
						OtherSettings: map[string]string{
							"setting1": "value1",
							"setting2": "value2",
						},
					},
				},
			},
		},
		{
			name: "no settings present",
			expected: &protobufs.ServerToAgent{
				ConnectionSettings: &protobufs.ConnectionSettingsOffers{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logr.Discard()
			instanceId := uuid.New()
			conn := &mockConnection{}
			bridge := tt.setSettings
			if bridge == nil {
				bridge = &mockBridgeAgent{}
			}
			agent := NewAgent(logger, instanceId, conn, bridge)
			response := &protobufs.ServerToAgent{}
			agent.calcConnectionSettings(response)

			if tt.expectedErr {
				assert.Nil(t, response.ConnectionSettings, "expected nil ConnectionSettings due to error")
			} else {
				assert.NotNil(t, response.ConnectionSettings, "expected non-nil ConnectionSettings")
				assert.Equal(t, tt.expected.ConnectionSettings.OwnMetrics, response.ConnectionSettings.OwnMetrics, "OwnMetrics should match")
				assert.Equal(t, tt.expected.ConnectionSettings.OwnTraces, response.ConnectionSettings.OwnTraces, "OwnTraces should match")
				assert.Equal(t, tt.expected.ConnectionSettings.OwnLogs, response.ConnectionSettings.OwnLogs, "OwnLogs should match")
				assert.Equal(t, tt.expected.ConnectionSettings.OtherConnections, response.ConnectionSettings.OtherConnections, "OtherConnections should match")
				assert.NotEmpty(t, response.ConnectionSettings.Hash, "Hash should not be empty")
			}
		})
	}
}

type mockBridgeAgent struct {
	ownMetrics       *protobufs.TelemetryConnectionSettings
	ownTraces        *protobufs.TelemetryConnectionSettings
	ownLogs          *protobufs.TelemetryConnectionSettings
	otherConnections map[string]*protobufs.OtherConnectionSettings
}

func (m *mockBridgeAgent) GetOwnMetricsSettings() *protobufs.TelemetryConnectionSettings {
	return m.ownMetrics
}

func (m *mockBridgeAgent) GetOwnTracesSettings() *protobufs.TelemetryConnectionSettings {
	return m.ownTraces
}

func (m *mockBridgeAgent) GetOwnLogsSettings() *protobufs.TelemetryConnectionSettings {
	return m.ownLogs
}

func (m *mockBridgeAgent) GetOtherConnectionSettings() map[string]*protobufs.OtherConnectionSettings {
	return m.otherConnections
}

func Test_CalcConnectionSettings_HashUniqueness(t *testing.T) {
	logger := logr.Discard()
	instanceId := uuid.New()
	conn := &mockConnection{}

	bridge1 := &mockBridgeAgent{
		ownMetrics: &protobufs.TelemetryConnectionSettings{
			DestinationEndpoint: "metrics-endpoint",
		},
	}
	agent1 := NewAgent(logger, instanceId, conn, bridge1)
	response1 := &protobufs.ServerToAgent{}
	agent1.calcConnectionSettings(response1)

	bridge2 := &mockBridgeAgent{
		ownTraces: &protobufs.TelemetryConnectionSettings{
			DestinationEndpoint: "traces-endpoint",
		},
	}
	agent2 := NewAgent(logger, instanceId, conn, bridge2)
	response2 := &protobufs.ServerToAgent{}
	agent2.calcConnectionSettings(response2)

	bridge3 := &mockBridgeAgent{}
	agent3 := NewAgent(logger, instanceId, conn, bridge3)
	response3 := &protobufs.ServerToAgent{}
	agent3.calcConnectionSettings(response3)

	assert.NotEqual(t, response1.ConnectionSettings.Hash, response2.ConnectionSettings.Hash)
	assert.NotEqual(t, response1.ConnectionSettings.Hash, response3.ConnectionSettings.Hash)
	assert.NotEqual(t, response2.ConnectionSettings.Hash, response3.ConnectionSettings.Hash)
}
