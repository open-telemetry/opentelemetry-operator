package opampsrv

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"

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
	marshalled, err := json.Marshal(converted)
	if err != nil {
		srv.logger.Errorf(request.Context(), "failed to marshal: %v", err)
		writer.WriteHeader(503)
		return
	}
	writer.Write(marshalled)
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
	marshalled, err := json.Marshal(agent)
	if err != nil {
		srv.logger.Errorf(request.Context(), "failed to marshal: %v", err)
		writer.WriteHeader(503)
		return
	}
	writer.Write(marshalled)

}
