// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/oklog/ulid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/logger"
)

type Server interface {
	GetConfigurations() map[uuid.UUID]*protobufs.EffectiveConfig
	GetHealth() map[uuid.UUID]*protobufs.ComponentHealth
	HasUpdates() <-chan struct{}
}

var _ Server = &OpAMPProxy{}

type OpAMPProxy struct {
	server.OpAMPServer

	// endpoint for service
	endpoint string

	// logger
	logger logr.Logger

	// updatesChan indicates if there any new agents
	updatesChan chan struct{}
	// internal state
	mux        sync.RWMutex
	agentsById map[uuid.UUID]*Agent
	// connections map is required because that's the only way we know to remove an agent.
	connections map[types.Connection]map[uuid.UUID]bool
}

func NewOpAMPProxy(l logr.Logger, endpoint string) *OpAMPProxy {
	return &OpAMPProxy{
		logger:      l,
		OpAMPServer: server.New(logger.NewLogger(l)),
		endpoint:    endpoint,
		agentsById:  map[uuid.UUID]*Agent{},
		connections: map[types.Connection]map[uuid.UUID]bool{},
		updatesChan: make(chan struct{}, 1),
	}
}

func (s *OpAMPProxy) Start() error {
	settings := server.StartSettings{
		Settings: server.Settings{
			CustomCapabilities: []string{},
			Callbacks: server.CallbacksStruct{
				OnConnectingFunc: func(request *http.Request) types.ConnectionResponse {
					return types.ConnectionResponse{
						Accept: true,
						ConnectionCallbacks: server.ConnectionCallbacksStruct{
							OnMessageFunc:         s.onMessage,
							OnConnectionCloseFunc: s.onDisconnect,
						},
					}
				},
			},
		},
		ListenEndpoint: s.endpoint,
		HTTPMiddleware: otelhttp.NewMiddleware("/v1/opamp"),
	}
	// TODO: In the future we will probably want some TLS configuration.
	// tlsConfig, err := internal.CreateServerTLSConfig(
	// 	"../../certs/certs/ca.cert.pem",
	// 	"../../certs/server_certs/server.cert.pem",
	// 	"../../certs/server_certs/server.key.pem",
	// )
	// if err != nil {
	// 	srv.logger.Debugf(context.Background(), "Could not load TLS config, working without TLS: %v", err.Error())
	// }
	// settings.TLSConfig = tlsConfig
	s.logger.Info("starting opamp server", "address", s.endpoint)
	return s.OpAMPServer.Start(settings)
}

func (s *OpAMPProxy) Stop(ctx context.Context) error {
	return s.OpAMPServer.Stop(ctx)
}

func (s *OpAMPProxy) onDisconnect(conn types.Connection) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for instanceId := range s.connections[conn] {
		delete(s.agentsById, instanceId)
	}
	delete(s.connections, conn)
	// Tell listeners to get updates.
	s.updatesChan <- struct{}{}
}

func (s *OpAMPProxy) onMessage(ctx context.Context, conn types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	// Start building the response.
	response := &protobufs.ServerToAgent{}

	instanceId, err := getInstanceId(msg)
	if err != nil {
		s.logger.Error(err, "failed to get instance id")
		response.ErrorResponse = &protobufs.ServerErrorResponse{
			ErrorMessage: err.Error(),
		}
		return response
	}
	s.logger.V(5).Info("received message", "instance ID", instanceId)

	agentUpdated := false
	s.mux.Lock()
	if _, ok := s.agentsById[instanceId]; !ok {
		s.agentsById[instanceId] = NewAgent(s.logger.WithValues("instanceId", instanceId.String()), instanceId, conn)
		// Ensure the Agent's instance id is associated with the connection.
		if s.connections[conn] == nil {
			s.connections[conn] = map[uuid.UUID]bool{}
		}
		s.connections[conn][instanceId] = true
		agentUpdated = true
	}
	agentUpdated = s.agentsById[instanceId].UpdateStatus(msg, response) || agentUpdated
	s.mux.Unlock()
	if agentUpdated {
		s.updatesChan <- struct{}{}
	}
	// Send the response back to the Agent.
	return response
}

// GetConfigurations implements Server.
func (s *OpAMPProxy) GetConfigurations() map[uuid.UUID]*protobufs.EffectiveConfig {
	s.mux.RLock()
	defer s.mux.RUnlock()
	toReturn := make(map[uuid.UUID]*protobufs.EffectiveConfig, len(s.agentsById))
	for i, agent := range s.agentsById {
		toReturn[i] = agent.GetConfiguration()
	}
	return toReturn
}

// GetHealth implements Server.
func (s *OpAMPProxy) GetHealth() map[uuid.UUID]*protobufs.ComponentHealth {
	s.mux.RLock()
	defer s.mux.RUnlock()
	toReturn := make(map[uuid.UUID]*protobufs.ComponentHealth, len(s.agentsById))
	for i, agent := range s.agentsById {
		toReturn[i] = agent.GetHealth()
	}
	return toReturn
}

// HasUpdates implements Server.
func (s *OpAMPProxy) HasUpdates() <-chan struct{} {
	return s.updatesChan
}

func getInstanceId(msg *protobufs.AgentToServer) (uuid.UUID, error) {
	var instanceId uuid.UUID

	if len(msg.InstanceUid) == 26 {
		// This is an old-style ULID.
		u, err := ulid.Parse(string(msg.InstanceUid))
		if err != nil {
			return instanceId, err
		}
		instanceId = uuid.UUID(u)
	} else if len(msg.InstanceUid) == 16 {
		// This is a 16 byte, new style UID.
		instanceId = uuid.UUID(msg.InstanceUid)
	} else {
		return instanceId, errors.New("invalid length of msg.InstanceUid")
	}
	return instanceId, nil
}
