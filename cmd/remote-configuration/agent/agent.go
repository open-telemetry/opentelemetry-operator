package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/operator"
	"gopkg.in/yaml.v3"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/config"

	"github.com/oklog/ulid/v2"
	"go.uber.org/multierr"

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

	// A set of the applied object keys (name/namespace)
	appliedKeys map[string]bool
	startTime   uint64

	instanceId ulid.ULID

	agentDescription *protobufs.AgentDescription

	opampClient client.OpAMPClient

	remoteConfigStatus *protobufs.RemoteConfigStatus

	metricReporter *MetricReporter
	config         config.Config
	applier        operator.ConfigApplier
	lastHash       []byte
}

func NewAgent(logger types.Logger, applier operator.ConfigApplier, config config.Config, agentType string, agentVersion string) *Agent {
	agent := &Agent{
		config:       config,
		applier:      applier,
		logger:       logger,
		agentType:    agentType,
		agentVersion: agentVersion,
		appliedKeys:  map[string]bool{},
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

func (agent *Agent) getNameAndNamespace(key string) (string, string, error) {
	s := strings.Split(key, "/")
	// We expect map keys to be of the form name/namespace
	if len(s) != 2 {
		return "", "", errors.New("invalid key")
	}
	return s[0], s[1], nil
}

func (agent *Agent) makeKeyFromNameNamespace(name string, namespace string) string {
	return fmt.Sprintf("%s/%s", name, namespace)
}

func (agent *Agent) composeEffectiveConfig() *protobufs.EffectiveConfig {
	effectiveConfig := &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: nil,
		},
	}
	instances, err := agent.applier.ListInstances()
	if err != nil {
		agent.logger.Errorf("couldn't list instances", err)
		return effectiveConfig
	}
	instanceMap := map[string]*protobufs.AgentConfigFile{}
	for _, instance := range instances {
		marshalled, err := yaml.Marshal(instance)
		if err != nil {
			agent.logger.Errorf("couldn't marshal collector configuration", err)
			continue
		}
		mapKey := agent.makeKeyFromNameNamespace(instance.GetName(), instance.GetNamespace())
		instanceMap[mapKey] = &protobufs.AgentConfigFile{
			Body:        marshalled,
			ContentType: "yaml",
		}
	}
	effectiveConfig.ConfigMap.ConfigMap = instanceMap
	return effectiveConfig
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
func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (bool, error) {
	if config == nil {
		return false, nil
	}
	if bytes.Equal(agent.lastHash, config.ConfigHash) {
		return false, nil
	}
	var multiErr error
	for key, file := range config.Config.GetConfigMap() {
		if len(key) == 0 || len(file.Body) == 0 {
			continue
		}
		name, namespace, err := agent.getNameAndNamespace(key)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			continue
		}
		err = agent.applier.Apply(name, namespace, file)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			continue
		}
		agent.appliedKeys[key] = true
	}
	if multiErr != nil {
		return false, multiErr
	}
	agent.lastHash = config.ConfigHash
	return true, nil
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
			setErr := agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
				ErrorMessage:         err.Error(),
			})
			if setErr != nil {
				return
			}
		} else {
			setErr := agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
			})
			if setErr != nil {
				return
			}
		}
	}

	// TODO: figure out why metrics aren't working
	//if msg.OwnMetricsConnSettings != nil {
	//	agent.initMeter(msg.OwnMetricsConnSettings)
	//}

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
