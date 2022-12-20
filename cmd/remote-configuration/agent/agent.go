package agent

import (
	"context"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/config"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

const localConfig = `
exporters:
  otlp:
    endpoint: localhost:1111

receivers:
  otlp:
    protocols:
      grpc: {}
      http: {}

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [otlp]
`

type Agent struct {
	logger types.Logger

	agentType    string
	agentVersion string

	effectiveConfig string
	startTime       uint64

	instanceId ulid.ULID

	agentDescription *protobufs.AgentDescription

	opampClient client.OpAMPClient

	remoteConfigStatus *protobufs.RemoteConfigStatus

	metricReporter *MetricReporter
	config         config.Config
}

func NewAgent(logger types.Logger, config config.Config, agentType string, agentVersion string) *Agent {
	agent := &Agent{
		config:       config,
		logger:       logger,
		agentType:    agentType,
		agentVersion: agentVersion,
	}

	agent.createAgentIdentity()
	agent.logger.Debugf("Agent starting, id=%v, type=%s, version=%s.",
		agent.instanceId.String(), agentType, agentVersion)
	agent.opampClient = agent.createClient()

	return agent
}

func (agent *Agent) createClient() client.OpAMPClient {
	if agent.config.Protocol == "http" {
		return client.NewHTTP(agent.logger)
	}
	return client.NewWebSocket(agent.logger)
}

func (agent *Agent) getHealth() *protobufs.AgentHealth {
	return &protobufs.AgentHealth{
		Healthy:           true,
		StartTimeUnixNano: agent.startTime,
		LastError:         "",
	}
}

func (agent *Agent) Start() error {
	agent.startTime = uint64(time.Now().UnixNano())
	settings := types.StartSettings{
		OpAMPServerURL: agent.config.Endpoint,
		//TLSConfig:      &tls.Config{InsecureSkipVerify: true},
		InstanceUid: agent.instanceId.String(),
		Callbacks: types.CallbacksStruct{
			OnConnectFunc: func() {
				agent.logger.Debugf("Connected to the server.")
			},
			OnConnectFailedFunc: func(err error) {
				agent.logger.Errorf("Failed to connect to the server: %v", err)
			},
			OnErrorFunc: func(err *protobufs.ServerErrorResponse) {
				agent.logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
			},
			SaveRemoteConfigStatusFunc: func(_ context.Context, status *protobufs.RemoteConfigStatus) {
				agent.remoteConfigStatus = status
			},
			GetEffectiveConfigFunc: func(ctx context.Context) (*protobufs.EffectiveConfig, error) {
				return agent.composeEffectiveConfig(), nil
			},
			OnMessageFunc: agent.onMessage,
		},
		RemoteConfigStatus:    agent.remoteConfigStatus,
		PackagesStateProvider: nil,
		Capabilities:          agent.config.GetCapabilities(),
	}
	err := agent.opampClient.SetAgentDescription(agent.agentDescription)
	if err != nil {
		return err
	}
	err = agent.opampClient.SetHealth(agent.getHealth())
	if err != nil {
		return err
	}

	agent.logger.Debugf("Starting OpAMP client...")

	err = agent.opampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	agent.logger.Debugf("OpAMP Client started.")

	return nil
}

func (agent *Agent) createAgentIdentity() {
	// Generate instance id.
	entropy := ulid.Monotonic(rand.New(rand.NewSource(0)), 0)
	agent.instanceId = ulid.MustNew(ulid.Timestamp(time.Now()), entropy)

	hostname, _ := os.Hostname()

	// Create Agent description.
	agent.agentDescription = &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			{
				Key: "service.name",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: agent.agentType},
				},
			},
			{
				Key: "service.version",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: agent.agentVersion},
				},
			},
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			{
				Key: "os.family",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{
						StringValue: runtime.GOOS,
					},
				},
			},
			{
				Key: "host.name",
				Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{
						StringValue: hostname,
					},
				},
			},
		},
	}
}

func (agent *Agent) updateAgentIdentity(instanceId ulid.ULID) {
	agent.logger.Debugf("Agent identify is being changed from id=%v to id=%v",
		agent.instanceId.String(),
		instanceId.String())
	agent.instanceId = instanceId

	if agent.metricReporter != nil {
		// TODO: reinit or update meter (possibly using a single function to update all own connection settings
		// or with having a common resource factory or so)
	}
}

func (agent *Agent) composeEffectiveConfig() *protobufs.EffectiveConfig {
	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"": {Body: []byte(agent.effectiveConfig)},
			},
		},
	}
}

func (agent *Agent) initMeter(settings *protobufs.TelemetryConnectionSettings) {
	reporter, err := NewMetricReporter(agent.logger, settings, agent.agentType, agent.agentVersion, agent.instanceId)
	if err != nil {
		agent.logger.Errorf("Cannot collect metrics: %v", err)
		return
	}

	prevReporter := agent.metricReporter

	agent.metricReporter = reporter

	if prevReporter != nil {
		prevReporter.Shutdown()
	}
	return
}

// Take the remote config, layer it over existing, done
func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (configChanged bool, err error) {
	if config == nil {
		return false, nil
	}
	return false, nil
}

func (agent *Agent) Shutdown() {
	agent.logger.Debugf("Agent shutting down...")
	if agent.opampClient != nil {
		_ = agent.opampClient.Stop(context.Background())
	}
}

func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {
	configChanged := false
	if msg.RemoteConfig != nil {
		var err error
		configChanged, err = agent.applyRemoteConfig(msg.RemoteConfig)
		if err != nil {
			agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
				ErrorMessage:         err.Error(),
			})
		} else {
			agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
			})
		}
	}

	if msg.OwnMetricsConnSettings != nil {
		agent.initMeter(msg.OwnMetricsConnSettings)
	}

	if msg.AgentIdentification != nil {
		newInstanceId, err := ulid.Parse(msg.AgentIdentification.NewInstanceUid)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
		agent.updateAgentIdentity(newInstanceId)
	}

	if configChanged {
		err := agent.opampClient.UpdateEffectiveConfig(ctx)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
	}
}
