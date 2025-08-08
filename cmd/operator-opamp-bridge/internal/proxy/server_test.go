// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"net"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpAMPProxy_StartStop(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	err := proxyServer.Start()
	require.NoError(t, err, "should be able to start the server")

	err = proxyServer.Stop(context.Background())
	require.NoError(t, err, "should be able to stop the server")
}

func TestOpAMPProxy_OnMessage(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	instanceId := uuid.New()
	conn := &mockConnection{}
	msg := &protobufs.AgentToServer{
		InstanceUid: instanceId[:],
	}

	response := proxyServer.onMessage(context.Background(), conn, msg)
	require.NotNil(t, response, "response should not be nil")
}

func TestOpAMPProxy_OnDisconnect(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	instanceId := uuid.New()
	conn := &mockConnection{}
	proxyServer.agentsById[instanceId] = NewAgent(logger, instanceId, conn)
	proxyServer.connections[conn] = map[uuid.UUID]bool{instanceId: true}

	proxyServer.onDisconnect(conn)

	assert.Empty(t, proxyServer.agentsById, "agentsById should be empty")
	assert.Empty(t, proxyServer.connections, "connections should be empty")
	assert.Empty(t, proxyServer.agentsByHostName, "agentsByHostName should be empty")
}

func TestOpAMPProxy_GetConfigurations(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn)
	proxyServer.agentsById[instanceId] = agent

	configs := proxyServer.GetConfigurations()
	require.NotNil(t, configs, "configs should not be nil")
	assert.Contains(t, configs, instanceId, "configs should contain the instance ID")
}

func TestOpAMPProxy_GetHealth(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	instanceId := uuid.New()
	conn := &mockConnection{}
	agent := NewAgent(logger, instanceId, conn)
	proxyServer.agentsById[instanceId] = agent

	healths := proxyServer.GetHealth()
	require.NotNil(t, healths, "healths should not be nil")
	assert.Contains(t, healths, instanceId, "healths should contain the instance ID")
}

func TestOpAMPProxy_GetAgentsByHostname(t *testing.T) {
	logger := logr.Discard()
	endpoint := "localhost:4321"
	proxyServer := NewOpAMPProxy(logger, endpoint)

	instanceId := uuid.New()
	proxyServer.agentsByHostName["example"] = instanceId

	byHostname := proxyServer.GetAgentsByHostname()
	require.NotNil(t, byHostname, "byHostname should not be nil")
	id, ok := byHostname["example"]
	assert.True(t, ok, "map should contain example key")
	assert.Equal(t, instanceId, id)
}

func TestGetInstanceId(t *testing.T) {
	tests := []struct {
		name        string
		instanceUid []byte
		wantErr     bool
	}{
		{
			name:        "valid ULID",
			instanceUid: []byte("01F8MECHZX3TBDSZ7XRADM79XE"),
			wantErr:     false,
		},
		{
			name: "valid UUID",
			instanceUid: func() []byte {
				bytes, _ := uuid.New().MarshalBinary()
				return bytes
			}(),
			wantErr: false,
		},
		{
			name:        "invalid length",
			instanceUid: []byte("invalid"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getInstanceId(&protobufs.AgentToServer{InstanceUid: tt.instanceUid})
			if (err != nil) != tt.wantErr {
				t.Errorf("getInstanceId() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockConnection struct{}

// Connection implements types.Connection.
func (m *mockConnection) Connection() net.Conn {
	panic("unimplemented")
}

// Disconnect implements types.Connection.
func (m *mockConnection) Disconnect() error {
	panic("unimplemented")
}

func (m *mockConnection) Send(ctx context.Context, msg *protobufs.ServerToAgent) error {
	return nil
}

func (m *mockConnection) Close() error {
	return nil
}
