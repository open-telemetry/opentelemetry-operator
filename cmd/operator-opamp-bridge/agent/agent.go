// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"bytes"
	"context"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/metrics"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"

	"github.com/oklog/ulid/v2"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type Agent struct {
	logger types.Logger

	appliedKeys map[collectorKey]bool
	startTime   uint64
	lastHash    []byte

	instanceId         ulid.ULID
	agentDescription   *protobufs.AgentDescription
	remoteConfigStatus *protobufs.RemoteConfigStatus

	opampClient         client.OpAMPClient
	metricReporter      *metrics.MetricReporter
	config              config.Config
	applier             operator.ConfigApplier
	remoteConfigEnabled bool
}

func NewAgent(logger types.Logger, applier operator.ConfigApplier, config config.Config, opampClient client.OpAMPClient) *Agent {
	agent := &Agent{
		config:              config,
		applier:             applier,
		logger:              logger,
		appliedKeys:         map[collectorKey]bool{},
		instanceId:          config.GetNewInstanceId(),
		agentDescription:    config.GetDescription(),
		remoteConfigEnabled: config.RemoteConfigEnabled(),
		opampClient:         opampClient,
	}

	agent.logger.Debugf("Agent created, id=%v, type=%s, version=%s.",
		agent.instanceId.String(), config.GetAgentType(), config.GetAgentVersion())

	return agent
}

// TODO: Something should run on a schedule to set the health of the OpAMP client.
func (agent *Agent) getHealth() *protobufs.AgentHealth {
	return &protobufs.AgentHealth{
		Healthy:           true,
		StartTimeUnixNano: agent.startTime,
		LastError:         "",
	}
}

// onConnect is called when an agent is successfully connected to a server.
func (agent *Agent) onConnect() {
	agent.logger.Debugf("Connected to the server.")
}

// onConnectFailed is called when an agent was unable to connect to a server.
func (agent *Agent) onConnectFailed(err error) {
	agent.logger.Errorf("Failed to connect to the server: %v", err)
}

// onError is called when an agent receives an error response from the server.
func (agent *Agent) onError(err *protobufs.ServerErrorResponse) {
	agent.logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
}

// saveRemoteConfigStatus receives a status from the server when the server sets a remote configuration.
func (agent *Agent) saveRemoteConfigStatus(_ context.Context, status *protobufs.RemoteConfigStatus) {
	agent.remoteConfigStatus = status
}

// Start sets up the callbacks for the OpAMP client and begins the client's connection to the server.
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

// updateAgentIdentity receives a new instanced Id from the remote server and updates the agent's instanceID field.
// The meter will be reinitialized by the onMessage function.
func (agent *Agent) updateAgentIdentity(instanceId ulid.ULID) {
	agent.logger.Debugf("Agent identity is being changed from id=%v to id=%v",
		agent.instanceId.String(),
		instanceId.String())
	agent.instanceId = instanceId
}

// getEffectiveConfig is called when a remote server needs to learn of the current effective configuration of each
// collector the agent is managing.
func (agent *Agent) getEffectiveConfig(ctx context.Context) (*protobufs.EffectiveConfig, error) {
	instances, err := agent.applier.ListInstances()
	if err != nil {
		agent.logger.Errorf("couldn't list instances", err)
		return nil, err
	}
	instanceMap := map[string]*protobufs.AgentConfigFile{}
	for _, instance := range instances {
		marshaled, err := yaml.Marshal(instance)
		if err != nil {
			agent.logger.Errorf("couldn't marshal collector configuration", err)
			return nil, err
		}
		mapKey := newCollectorKey(instance.GetName(), instance.GetNamespace())
		instanceMap[mapKey.String()] = &protobufs.AgentConfigFile{
			Body:        marshaled,
			ContentType: "yaml",
		}
	}
	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: instanceMap,
		},
	}, nil
}

// initMeter initializes a metric reporter instance for the agent to report runtime metrics to the
// configured destination. The settings received will be used to initialize a reporter, shutting down any previously
// running metrics reporting instances.
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

// applyRemoteConfig receives a remote configuration from a remote server of the following form:
//
//	map[name/namespace] -> collector CRD spec
//
// For every key in the received remote configuration, the agent attempts to apply it to the connected
// Kubernetes cluster. If an agent fails to apply a collector CRD, it will continue to the next entry. The agent will
// store the received configuration hash regardless of application status as per the OpAMP spec.
//
// INVARIANT: The caller must verify that config isn't nil _and_ the configuration has changed between calls.
func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (*protobufs.RemoteConfigStatus, error) {
	var multiErr error
	// Apply changes from the received config map
	for key, file := range config.Config.GetConfigMap() {
		if len(key) == 0 || len(file.Body) == 0 {
			continue
		}
		colKey, err := collectorKeyFromKey(key)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			continue
		}
		err = agent.applier.Apply(colKey.name, colKey.namespace, file)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			continue
		}
		agent.appliedKeys[colKey] = true
	}
	// Check if anything was deleted
	for collectorKey := range agent.appliedKeys {
		if _, ok := config.Config.GetConfigMap()[collectorKey.String()]; !ok {
			err := agent.applier.Delete(collectorKey.name, collectorKey.namespace)
			if err != nil {
				multiErr = multierr.Append(multiErr, err)
			}
		}
	}
	agent.lastHash = config.GetConfigHash()
	if multiErr != nil {
		return &protobufs.RemoteConfigStatus{
			LastRemoteConfigHash: agent.lastHash,
			Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
			ErrorMessage:         multiErr.Error(),
		}, multiErr
	}
	return &protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: agent.lastHash,
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
	}, nil
}

// Shutdown will stop the OpAMP client gracefully.
func (agent *Agent) Shutdown() {
	agent.logger.Debugf("Agent shutting down...")
	if agent.opampClient != nil {
		err := agent.opampClient.Stop(context.Background())
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
	}
	if agent.metricReporter != nil {
		agent.metricReporter.Shutdown()
	}
}

// onMessage is called when the client receives a new message from the connected OpAMP server. The agent is responsible
// for checking if it should apply a new remote configuration. The agent will also initialize metrics based on the
// settings received from the server. The agent is also able to update its identifier if it needs to.
func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {
	// If we received remote configuration, and it's not the same as the previously applied one
	if agent.remoteConfigEnabled && msg.RemoteConfig != nil && !bytes.Equal(agent.lastHash, msg.RemoteConfig.GetConfigHash()) {
		var err error
		status, err := agent.applyRemoteConfig(msg.RemoteConfig)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
		err = agent.opampClient.SetRemoteConfigStatus(status)
		if err != nil {
			agent.logger.Errorf(err.Error())
			return
		}
		err = agent.opampClient.UpdateEffectiveConfig(ctx)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
	}

	// The instance id is updated prior to the meter initialization so that the new meter will report using the updated
	// instanceId.
	if msg.AgentIdentification != nil {
		newInstanceId, err := ulid.Parse(msg.AgentIdentification.NewInstanceUid)
		if err != nil {
			agent.logger.Errorf(err.Error())
			return
		}
		agent.updateAgentIdentity(newInstanceId)
	}

	if msg.OwnMetricsConnSettings != nil {
		agent.initMeter(msg.OwnMetricsConnSettings)
	}
}
