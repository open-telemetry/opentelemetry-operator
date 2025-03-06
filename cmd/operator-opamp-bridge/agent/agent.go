// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/multierr"
	"k8s.io/utils/clock"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/metrics"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/operator"
)

type Agent struct {
	logger logr.Logger

	appliedKeys map[kubeResourceKey]bool
	clock       clock.Clock
	startTime   uint64
	lastHash    []byte

	instanceId         uuid.UUID
	agentDescription   *protobufs.AgentDescription
	remoteConfigStatus *protobufs.RemoteConfigStatus

	opampClient         client.OpAMPClient
	metricReporter      *metrics.MetricReporter
	config              *config.Config
	applier             operator.ConfigApplier
	remoteConfigEnabled bool

	done   chan struct{}
	ticker *time.Ticker
}

func NewAgent(logger logr.Logger, applier operator.ConfigApplier, config *config.Config, opampClient client.OpAMPClient) *Agent {
	var t *time.Ticker
	if config.HeartbeatInterval > 0 {
		t = time.NewTicker(config.HeartbeatInterval)
	}
	agent := &Agent{
		config:              config,
		applier:             applier,
		logger:              logger,
		appliedKeys:         map[kubeResourceKey]bool{},
		instanceId:          config.GetNewInstanceId(),
		agentDescription:    config.GetDescription(),
		remoteConfigEnabled: config.RemoteConfigEnabled(),
		opampClient:         opampClient,
		clock:               clock.RealClock{},
		done:                make(chan struct{}, 1),
		ticker:              t,
	}

	agent.logger.V(3).Info("Agent created",
		"instanceId", agent.instanceId.String(),
		"agentType", config.GetAgentType(),
		"agentVersion", config.GetAgentVersion())

	return agent
}

// getHealth is called every heartbeat interval to report health.
func (agent *Agent) getHealth() *protobufs.ComponentHealth {
	healthMap, err := agent.generateCollectorPoolHealth()
	if err != nil {
		return &protobufs.ComponentHealth{
			Healthy:           false,
			StartTimeUnixNano: agent.startTime,
			LastError:         err.Error(),
		}
	}
	statusTime, err := agent.getCurrentTimeUnixNano()
	if err != nil {
		return &protobufs.ComponentHealth{
			Healthy:           false,
			StartTimeUnixNano: agent.startTime,
			LastError:         err.Error(),
		}
	}
	return &protobufs.ComponentHealth{
		Healthy:            true,
		StartTimeUnixNano:  agent.startTime,
		StatusTimeUnixNano: statusTime,
		LastError:          "",
		ComponentHealthMap: healthMap,
	}
}

// generateCollectorPoolHealth allows the bridge to report the status of the collector pools it owns.
// TODO: implement enhanced health messaging using the collector's new healthcheck extension:
// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/26661
func (agent *Agent) generateCollectorPoolHealth() (map[string]*protobufs.ComponentHealth, error) {
	cols, err := agent.applier.ListInstances()
	if err != nil {
		return nil, err
	}
	healthMap := map[string]*protobufs.ComponentHealth{}
	for _, col := range cols {
		key := newKubeResourceKey(col.GetNamespace(), col.GetName())
		podMap, err := agent.generateCollectorHealth(agent.getCollectorSelector(col), col.GetNamespace())
		if err != nil {
			return nil, err
		}

		isPoolHealthy := true
		for _, pod := range podMap {
			isPoolHealthy = isPoolHealthy && pod.Healthy
		}
		podStartTime, err := timeToUnixNanoUnsigned(col.ObjectMeta.GetCreationTimestamp().Time)
		if err != nil {
			return nil, err
		}
		statusTime, err := agent.getCurrentTimeUnixNano()
		if err != nil {
			return nil, err
		}
		healthMap[key.String()] = &protobufs.ComponentHealth{
			StartTimeUnixNano:  podStartTime,
			StatusTimeUnixNano: statusTime,
			Status:             col.Status.Scale.StatusReplicas,
			ComponentHealthMap: podMap,
			Healthy:            isPoolHealthy,
		}
	}
	return healthMap, nil
}

// getCollectorSelector destructures the collectors scale selector if present, if uses the labelmap from the operator.
func (agent *Agent) getCollectorSelector(col v1beta1.OpenTelemetryCollector) map[string]string {
	if len(col.Status.Scale.Selector) > 0 {
		selMap := map[string]string{}
		for _, kvPair := range strings.Split(col.Status.Scale.Selector, ",") {
			kv := strings.Split(kvPair, "=")
			// skip malformed pairs
			if len(kv) != 2 {
				continue
			}
			selMap[kv[0]] = kv[1]
		}
		return selMap
	}
	return map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", col.GetNamespace(), col.GetName()),
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
	}
}

func (agent *Agent) generateCollectorHealth(selectorLabels map[string]string, namespace string) (map[string]*protobufs.ComponentHealth, error) {
	statusTime, err := agent.getCurrentTimeUnixNano()
	if err != nil {
		return nil, err
	}
	pods, err := agent.applier.GetCollectorPods(selectorLabels, namespace)
	if err != nil {
		return nil, err
	}
	healthMap := map[string]*protobufs.ComponentHealth{}
	for _, item := range pods.Items {
		key := newKubeResourceKey(item.GetNamespace(), item.GetName())
		healthy := true
		if item.Status.Phase != "Running" {
			healthy = false
		}
		var startTime uint64
		if item.Status.StartTime != nil {
			startTime, err = timeToUnixNanoUnsigned(item.Status.StartTime.Time)
			if err != nil {
				return nil, err
			}
		} else {
			healthy = false
		}
		healthMap[key.String()] = &protobufs.ComponentHealth{
			StartTimeUnixNano:  startTime,
			StatusTimeUnixNano: statusTime,
			Status:             string(item.Status.Phase),
			Healthy:            healthy,
		}
	}
	return healthMap, nil
}

// onConnect is called when an agent is successfully connected to a server.
func (agent *Agent) onConnect(ctx context.Context) {
	agent.logger.V(3).Info("Connected to the server.")
}

// onConnectFailed is called when an agent was unable to connect to a server.
func (agent *Agent) onConnectFailed(ctx context.Context, err error) {
	agent.logger.Error(err, "failed to connect to the server")
}

// onError is called when an agent receives an error response from the server.
func (agent *Agent) onError(ctx context.Context, err *protobufs.ServerErrorResponse) {
	agent.logger.Error(errors.New(err.GetErrorMessage()), "server returned an error response")
}

// saveRemoteConfigStatus receives a status from the server when the server sets a remote configuration.
func (agent *Agent) saveRemoteConfigStatus(_ context.Context, status *protobufs.RemoteConfigStatus) {
	agent.remoteConfigStatus = status
}

// Start sets up the callbacks for the OpAMP client and begins the client's connection to the server.
func (agent *Agent) Start() error {
	startTime, err := agent.getCurrentTimeUnixNano()
	if err != nil {
		return err
	}
	agent.startTime = startTime
	settings := types.StartSettings{
		OpAMPServerURL: agent.config.Endpoint,
		Header:         agent.config.Headers.ToHTTPHeader(),
		InstanceUid:    types.InstanceUid(agent.instanceId),
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
	err = agent.opampClient.SetAgentDescription(agent.agentDescription)
	if err != nil {
		return err
	}
	err = agent.opampClient.SetHealth(agent.getHealth())
	if err != nil {
		return err
	}

	agent.logger.V(3).Info("Starting OpAMP client...")

	err = agent.opampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	if agent.config.HeartbeatInterval > 0 {
		go agent.runHeartbeat()
	}

	agent.logger.V(3).Info("OpAMP Client started.")

	return nil
}

// runHeartbeat sets health on an interval to keep the connection active.
func (agent *Agent) runHeartbeat() {
	if agent.ticker == nil {
		agent.logger.Info("cannot run heartbeat without setting an interval for the ticker")
		return
	}
	for {
		select {
		case <-agent.ticker.C:
			agent.logger.V(4).Info("sending heartbeat")
			err := agent.opampClient.SetHealth(agent.getHealth())
			if err != nil {
				agent.logger.Error(err, "failed to heartbeat")
				return
			}
		case <-agent.done:
			agent.ticker.Stop()
			agent.logger.Info("stopping heartbeating")
			return
		}
	}
}

// updateAgentIdentity receives a new instanced Id from the remote server and updates the agent's instanceID field.
// The meter will be reinitialized by the onMessage function.
func (agent *Agent) updateAgentIdentity(instanceId uuid.UUID) {
	agent.logger.V(3).Info("Agent identity is being changed",
		"old instanceId", agent.instanceId.String(),
		"new instanceid", instanceId.String())
	agent.instanceId = instanceId
}

// getEffectiveConfig is called when a remote server needs to learn of the current effective configuration of each
// collector the agent is managing.
func (agent *Agent) getEffectiveConfig(ctx context.Context) (*protobufs.EffectiveConfig, error) {
	instances, err := agent.applier.ListInstances()
	if err != nil {
		agent.logger.Error(err, "failed to list instances")
		return nil, err
	}
	instanceMap := map[string]*protobufs.AgentConfigFile{}
	for _, instance := range instances {
		col := instance
		marshaled, err := yaml.Marshal(&col)
		if err != nil {
			agent.logger.Error(err, "failed to marhsal config")
			return nil, err
		}
		mapKey := newKubeResourceKey(instance.GetNamespace(), instance.GetName())
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
		agent.logger.Error(err, "failed to create metric reporter")
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
		colKey, err := kubeResourceFromKey(key)
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
	agent.logger.V(3).Info("Agent shutting down...")
	close(agent.done)
	if agent.opampClient != nil {
		err := agent.opampClient.Stop(context.Background())
		if err != nil {
			agent.logger.Error(err, "failed to stop client")
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
			agent.logger.Error(err, "failed to apply remote config")
		}
		err = agent.opampClient.SetRemoteConfigStatus(status)
		if err != nil {
			agent.logger.Error(err, "failed to set remote config status")
			return
		}
		err = agent.opampClient.UpdateEffectiveConfig(ctx)
		if err != nil {
			agent.logger.Error(err, "failed to update effective config")
		}
	}

	// The instance id is updated prior to the meter initialization so that the new meter will report using the updated
	// instanceId.
	if msg.AgentIdentification != nil {
		uid, err := uuid.FromBytes(msg.AgentIdentification.NewInstanceUid)
		if err != nil {
			agent.logger.Error(err, "couldn't parse instance UID")
			return
		}
		agent.updateAgentIdentity(uid)
	}

	if msg.OwnMetricsConnSettings != nil {
		agent.initMeter(msg.OwnMetricsConnSettings)
	}
}

// getCurrentTimeUnixNano returns the current time as a uint64, which the protocol expects.
func (agent *Agent) getCurrentTimeUnixNano() (uint64, error) {
	// technically this could be negative if the system time is set to before 1970-01-1
	// the proto demands this to be a nonnegative number, so in that case, just return 0
	return timeToUnixNanoUnsigned(agent.clock.Now())
}

// timeToUnixNanoUnsigned returns the number of nanoseconds elapsed from 1970-01-01 to the given time, but returns an
// error if the value is negative. OpAMP expects these values to be non-negative.
func timeToUnixNanoUnsigned(t time.Time) (uint64, error) {
	signedUnixNano := t.UnixNano()
	if signedUnixNano < 0 {
		return 0, fmt.Errorf("invalid system time, must be after 01-01-1970 due to OpAMP requirements: %v", t)
	}
	return uint64(signedUnixNano), nil
}
