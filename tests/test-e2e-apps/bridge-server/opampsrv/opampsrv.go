// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampsrv

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/open-telemetry/opentelemetry-operator/tests/test-e2e-apps/bridge-server/data"
)

type Server struct {
	opampSrv   server.OpAMPServer
	agents     *data.Agents
	logger     *Logger
	httpServer *http.Server
}

type remoteConfigRequest struct {
	Config      map[string]string `json:"config"`
	ContentType string            `json:"content_type,omitempty"`
}

func NewServer(agents *data.Agents) *Server {
	logger := &Logger{
		log.New(
			log.Default().Writer(),
			"[OPAMP] ",
			log.Default().Flags()|log.Lmsgprefix|log.Lmicroseconds,
		),
	}

	srv := &Server{
		agents: agents,
		logger: logger,
	}

	srv.opampSrv = server.New(logger)

	return srv
}

func (srv *Server) Start() {
	settings := server.StartSettings{
		Settings: server.Settings{
			Callbacks: server.CallbacksStruct{
				OnConnectingFunc: func(request *http.Request) types.ConnectionResponse {
					return types.ConnectionResponse{
						Accept: true,
						ConnectionCallbacks: server.ConnectionCallbacksStruct{
							OnMessageFunc:         srv.onMessage,
							OnConnectionCloseFunc: srv.onDisconnect,
						},
					}
				},
			},
		},
		ListenEndpoint: "0.0.0.0:4320",
		HTTPMiddleware: otelhttp.NewMiddleware("/v1/opamp"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/agents", srv.getAgents)
	mux.HandleFunc("/agents/push-config-to-agent", srv.pushConfigToAgent)
	mux.HandleFunc("/agents/", srv.getAgentById)
	srv.httpServer = &http.Server{
		Addr:    "0.0.0.0:4321",
		Handler: mux,
	}
	go func() {
		err := srv.httpServer.ListenAndServe()
		if err != nil {
			srv.logger.Errorf(context.Background(), "HTTP server start fail: %v", err.Error())
			os.Exit(1)
		}
	}()

	if err := srv.opampSrv.Start(settings); err != nil {
		srv.logger.Errorf(context.Background(), "OpAMP server start fail: %v", err.Error())
		os.Exit(1)
	}
}

func (srv *Server) Stop() {
	ctx := context.Background()
	srv.httpServer.Shutdown(ctx)
	srv.opampSrv.Stop(ctx)
}

func (srv *Server) onDisconnect(conn types.Connection) {
	srv.agents.RemoveConnection(conn)
}

func (srv *Server) onMessage(ctx context.Context, conn types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	// Start building the response.
	response := &protobufs.ServerToAgent{}

	var instanceId data.InstanceId
	if len(msg.InstanceUid) == 26 {
		// This is an old-style ULID.
		u, err := ulid.Parse(string(msg.InstanceUid))
		if err != nil {
			srv.logger.Errorf(ctx, "Cannot parse ULID %s: %v", string(msg.InstanceUid), err)
			return response
		}
		instanceId = data.InstanceId(u.Bytes())
	} else if len(msg.InstanceUid) == 16 {
		// This is a 16 byte, new style UID.
		if parsedId, err := uuid.FromBytes(msg.InstanceUid); err != nil {
			srv.logger.Errorf(ctx, "Cannot parse UUID %s: %v", msg.InstanceUid, err)
			return response
		} else {
			instanceId = data.InstanceId(parsedId)
		}
	} else {
		srv.logger.Errorf(ctx, "Invalid length of msg.InstanceUid")
		return response
	}

	agent := srv.agents.FindOrCreateAgent(instanceId, conn)

	// Process the status report and continue building the response.
	agent.UpdateStatus(msg, response)

	// Send the response back to the Agent.
	return response
}

func (srv *Server) getAgents(writer http.ResponseWriter, request *http.Request) {
	allAgents := srv.agents.GetAllAgentsReadonlyClone()
	converted := map[string]*data.Agent{}
	for id, agent := range allAgents {
		converted[uuid.UUID(id).String()] = agent
	}
	marshaled, err := json.Marshal(converted)
	if err != nil {
		srv.logger.Errorf(request.Context(), "failed to marshal: %v", err)
		writer.WriteHeader(503)
		return
	}
	writer.Write(marshaled)
}

func (srv *Server) getAgentById(writer http.ResponseWriter, request *http.Request) {
	// Define a regex to extract the agent ID from the URL
	re := regexp.MustCompile(`^/agents/([0-9a-z\-]+)$`)
	matches := re.FindStringSubmatch(request.URL.Path)
	if len(matches) == 0 {
		http.NotFound(writer, request)
		return
	}
	parsed, err := uuid.Parse(matches[1])
	if err != nil {
		http.Error(writer, "invalid uuid", http.StatusBadRequest)
		return
	}
	agent := srv.agents.FindAgent(data.InstanceId(parsed))
	marshaled, err := json.Marshal(agent)
	if err != nil {
		srv.logger.Errorf(request.Context(), "failed to marshal: %v", err)
		writer.WriteHeader(503)
		return
	}
	writer.Write(marshaled)
}

func (srv *Server) pushConfigToAgent(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := parseRemoteConfigRequest(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	allAgents := srv.agents.GetAllAgentsReadonlyClone()
	if len(allAgents) != 1 {
		http.Error(writer, "expected exactly one connected agent", http.StatusConflict)
		return
	}

	var agentId data.InstanceId
	for id := range allAgents {
		agentId = id
	}

	statusUpdated := make(chan struct{}, 1)
	srv.agents.SetCustomConfigForAgent(agentId, config, statusUpdated)

	select {
	case <-statusUpdated:
	case <-request.Context().Done():
		return
	case <-time.After(30 * time.Second):
		http.Error(writer, "timed out waiting for agent status update", http.StatusGatewayTimeout)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

func parseRemoteConfigRequest(request *http.Request) (*protobufs.AgentConfigMap, error) {
	var req remoteConfigRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		return nil, err
	}
	if len(req.Config) == 0 {
		return nil, errors.New("config must contain at least one entry")
	}

	configMap := make(map[string]*protobufs.AgentConfigFile, len(req.Config))
	for key, body := range req.Config {
		if key == "" {
			return nil, errors.New("config keys must be non-empty")
		}
		configMap[key] = &protobufs.AgentConfigFile{
			Body:        []byte(body),
			ContentType: req.ContentType,
		}
	}

	return &protobufs.AgentConfigMap{
		ConfigMap: configMap,
	}, nil
}
