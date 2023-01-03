package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/metrics"
	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/operator"

	"github.com/open-telemetry/opentelemetry-operator/cmd/remote-configuration/config"

	"github.com/oklog/ulid/v2"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type Agent struct {
	logger types.Logger

	// A set of the applied object keys (name/namespace)
	appliedKeys map[string]bool
	startTime   uint64
	lastHash    []byte

	instanceId         ulid.ULID
	agentDescription   *protobufs.AgentDescription
	remoteConfigStatus *protobufs.RemoteConfigStatus

	opampClient    client.OpAMPClient
	metricReporter *metrics.MetricReporter
	config         config.Config
	applier        operator.ConfigApplier
}

func NewAgent(logger types.Logger, applier operator.ConfigApplier, config config.Config, opampClient client.OpAMPClient) *Agent {
	agent := &Agent{
		config:           config,
		applier:          applier,
		logger:           logger,
		appliedKeys:      map[string]bool{},
		instanceId:       config.GetNewInstanceId(),
		agentDescription: config.GetDescription(),
		opampClient:      opampClient,
	}

	agent.logger.Debugf("Agent starting, id=%v, type=%s, version=%s.",
		agent.instanceId.String(), config.GetAgentType(), config.GetAgentVersion())

	return agent
}

func (agent *Agent) getHealth() *protobufs.AgentHealth {
	return &protobufs.AgentHealth{
		Healthy:           true,
		StartTimeUnixNano: agent.startTime,
		LastError:         "",
	}
}

func (agent *Agent) onConnect() {
	agent.logger.Debugf("Connected to the server.")
}

func (agent *Agent) onConnectFailed(err error) {
	agent.logger.Errorf("Failed to connect to the server: %v", err)
}

func (agent *Agent) onError(err *protobufs.ServerErrorResponse) {
	agent.logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
}

func (agent *Agent) saveRemoteConfigStatus(_ context.Context, status *protobufs.RemoteConfigStatus) {
	agent.remoteConfigStatus = status
}

func (agent *Agent) Start() error {
	agent.startTime = uint64(time.Now().UnixNano())
	settings := types.StartSettings{
		OpAMPServerURL: agent.config.Endpoint,
		InstanceUid:    agent.instanceId.String(),
		Callbacks: types.CallbacksStruct{
			OnConnectFunc:              agent.onConnect,
			OnConnectFailedFunc:        agent.onConnectFailed,
			OnErrorFunc:                agent.onError,
			SaveRemoteConfigStatusFunc: agent.saveRemoteConfigStatus,
			GetEffectiveConfigFunc:     agent.getEffectiveConfig,
			OnMessageFunc:              agent.onMessage,
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

func (agent *Agent) getEffectiveConfig(ctx context.Context) (*protobufs.EffectiveConfig, error) {
	instances, err := agent.applier.ListInstances()
	if err != nil {
		agent.logger.Errorf("couldn't list instances", err)
		return nil, err
	}
	instanceMap := map[string]*protobufs.AgentConfigFile{}
	for _, instance := range instances {
		marshalled, err := yaml.Marshal(instance)
		if err != nil {
			agent.logger.Errorf("couldn't marshal collector configuration", err)
			return nil, err
		}
		mapKey := agent.makeKeyFromNameNamespace(instance.GetName(), instance.GetNamespace())
		instanceMap[mapKey] = &protobufs.AgentConfigFile{
			Body:        marshalled,
			ContentType: "yaml",
		}
	}
	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: instanceMap,
		},
	}, nil
}

func (agent *Agent) initMeter(settings *protobufs.TelemetryConnectionSettings) {
	reporter, err := metrics.NewMetricReporter(agent.logger, settings, agent.config.GetAgentType(), agent.config.GetAgentVersion(), agent.instanceId)
	if err != nil {
		agent.logger.Errorf("Cannot collect metrics: %v", err)
		return
	}

	if agent.metricReporter != nil {
		agent.metricReporter.Shutdown()
	}
	agent.metricReporter = reporter
}

// Take the remote config, layer it over existing, done
// INVARIANT: The caller must verify that config isn't nil _and_ the configuration has changed between calls
func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (*protobufs.RemoteConfigStatus, error) {
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
	for collectorKey := range agent.appliedKeys {
		name, namespace, err := agent.getNameAndNamespace(collectorKey)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			continue
		}
		if _, ok := config.Config.GetConfigMap()[collectorKey]; !ok {
			err = agent.applier.Delete(name, namespace)
			if err != nil {
				multiErr = multierr.Append(multiErr, err)
			}
		}
	}
	if multiErr != nil {
		return &protobufs.RemoteConfigStatus{
			LastRemoteConfigHash: config.GetConfigHash(),
			Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
			ErrorMessage:         multiErr.Error(),
		}, multiErr
	}
	agent.lastHash = config.ConfigHash
	return &protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: config.ConfigHash,
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
	}, nil
}

func (agent *Agent) Shutdown() {
	agent.logger.Debugf("Agent shutting down...")
	if agent.opampClient != nil {
		_ = agent.opampClient.Stop(context.Background())
	}
}

func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {

	// If we received remote configuration, and it's not the same as the previously applied one
	if msg.RemoteConfig != nil && !bytes.Equal(agent.lastHash, msg.RemoteConfig.GetConfigHash()) {
		var err error
		status, err := agent.applyRemoteConfig(msg.RemoteConfig)
		setErr := agent.opampClient.SetRemoteConfigStatus(status)
		if setErr != nil {
			return
		}
		err = agent.opampClient.UpdateEffectiveConfig(ctx)
		if err != nil {
			agent.logger.Errorf(err.Error())
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
}
