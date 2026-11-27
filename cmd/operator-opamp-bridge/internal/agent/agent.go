// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"
	"k8s.io/utils/clock"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/metrics"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

type Agent struct {
	logger logr.Logger

	appliedKeys map[string]bool
	clock       clock.Clock
	startTime   uint64
	lastHash    []byte

	instanceId         uuid.UUID
	agentDescription   *protobufs.AgentDescription
	remoteConfigStatus *protobufs.RemoteConfigStatus

	opampClient client.OpAMPClient
	proxy       proxy.Server
	// metricReporter is nil until the OpAMP server offers own-metrics connection settings. It is read from the
	// heartbeat, proxy and OpAMP client goroutines, so it is stored atomically.
	metricReporter atomic.Pointer[metrics.MetricReporter]
	// ownMetricsSettings are the settings the current metricReporter was built from. They are only accessed from
	// onMessage, which the OpAMP client invokes sequentially.
	ownMetricsSettings  *protobufs.TelemetryConnectionSettings
	config              *config.Config
	applier             operator.ConfigApplier
	remoteConfigEnabled bool

	// newReporter builds the metric reporter for the settings offered by the server. It is a field so tests can
	// replace it with a reporter that does not export over the network.
	newReporter func(logr.Logger, *protobufs.TelemetryConnectionSettings, string, string, uuid.UUID) (*metrics.MetricReporter, error)

	done   chan struct{}
	ticker *time.Ticker
}

func NewAgent(logger logr.Logger, applier operator.ConfigApplier, cfg *config.Config, opampClient client.OpAMPClient, p proxy.Server) *Agent {
	var t *time.Ticker
	if cfg.HeartbeatInterval > 0 {
		t = time.NewTicker(cfg.HeartbeatInterval)
	}
	agent := &Agent{
		config:              cfg,
		applier:             applier,
		proxy:               p,
		logger:              logger,
		appliedKeys:         map[string]bool{},
		instanceId:          cfg.GetInstanceId(),
		agentDescription:    cfg.GetDescription(),
		remoteConfigEnabled: cfg.RemoteConfigEnabled(),
		opampClient:         opampClient,
		clock:               clock.RealClock{},
		newReporter:         metrics.NewMetricReporter,
		done:                make(chan struct{}, 1),
		ticker:              t,
	}

	agent.logger.V(3).Info("Agent created",
		"instanceId", agent.instanceId.String(),
		"agentType", cfg.GetAgentType(),
		"agentVersion", cfg.GetAgentVersion())

	return agent
}

// getHealth is called every heartbeat interval to report health.
func (agent *Agent) getHealth() *protobufs.ComponentHealth {
	health, err := agent.applier.GetHealth()
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
	componentHealth, err := agent.componentHealthFromInternal(health, statusTime, true)
	if err != nil {
		return &protobufs.ComponentHealth{
			Healthy:           false,
			StartTimeUnixNano: agent.startTime,
			LastError:         err.Error(),
		}
	}
	agent.recordCollectorHealth(health)
	agent.addProxyHealth(componentHealth)
	return componentHealth
}

// recordCollectorHealth reports how many of the collectors managed by the bridge are healthy.
func (agent *Agent) recordCollectorHealth(health operator.Health) {
	var healthy, unhealthy int64
	for _, collector := range health.Children {
		if collector.Healthy {
			healthy++
		} else {
			unhealthy++
		}
	}
	agent.reporter().RecordCollectors(context.Background(), healthy, unhealthy)
}

// reporter returns the current metric reporter, which may be nil. All of its Record methods are nil-safe.
func (agent *Agent) reporter() *metrics.MetricReporter {
	return agent.metricReporter.Load()
}

// componentHealthFromInternal converts the bridge's internal recursive Health tree into the OpAMP protobuf shape.
// statusTime is applied to every component in the tree; root controls whether the bridge start time or the
// component-specific start time is reported.
func (agent *Agent) componentHealthFromInternal(health operator.Health, statusTime uint64, root bool) (*protobufs.ComponentHealth, error) {
	startTime := uint64(0)
	if root {
		startTime = agent.startTime
	} else if !health.StartTime.IsZero() {
		var err error
		startTime, err = timeToUnixNanoUnsigned(health.StartTime)
		if err != nil {
			return nil, err
		}
	}
	componentHealthMap := map[string]*protobufs.ComponentHealth{}
	for key, child := range health.Children {
		childHealth, err := agent.componentHealthFromInternal(child, statusTime, false)
		if err != nil {
			return nil, err
		}
		componentHealthMap[key] = childHealth
	}
	return &protobufs.ComponentHealth{
		Healthy:            health.Healthy,
		StartTimeUnixNano:  startTime,
		StatusTimeUnixNano: statusTime,
		Status:             health.Status,
		LastError:          health.LastError,
		ComponentHealthMap: componentHealthMap,
	}, nil
}

// addProxyHealth merges directly connected collector agents into the reported health tree.
// rootHealth is mutated in place: proxy agents are attached beneath matching pods by hostname, and any
// unmatched proxy agents are reported as root-level components.
func (agent *Agent) addProxyHealth(rootHealth *protobufs.ComponentHealth) {
	proxyHealth := agent.proxy.GetHealth()
	agent.reporter().RecordProxyAgents(context.Background(), int64(len(proxyHealth)))
	agentsByHostName := agent.proxy.GetAgentsByHostname()
	proxiesUsed := make(map[uuid.UUID]struct{}, len(agentsByHostName))
	for collectorName, collector := range rootHealth.ComponentHealthMap {
		namespace, _, ok := strings.Cut(collectorName, "/")
		if !ok {
			continue
		}
		for podName, pod := range collector.ComponentHealthMap {
			// we need to remove the prefix here as we don't have the namespace from the hostname.
			podNameWithoutNamespace := strings.TrimPrefix(podName, namespace+"/")
			if uid, ok := agentsByHostName[podNameWithoutNamespace]; ok {
				if pod.ComponentHealthMap == nil {
					pod.ComponentHealthMap = map[string]*protobufs.ComponentHealth{}
				}
				pod.ComponentHealthMap[uid.String()] = proxyHealth[uid]
				proxiesUsed[uid] = struct{}{}
			}
		}
	}
	for instance, instanceHealth := range proxyHealth {
		if _, ok := proxiesUsed[instance]; ok {
			continue
		}
		rootHealth.ComponentHealthMap[instance.String()] = instanceHealth
	}
}

// onConnect is called when an agent is successfully connected to a server.
func (agent *Agent) onConnect(ctx context.Context) {
	agent.logger.V(3).Info("Connected to the server.")
	agent.reporter().RecordConnectionAttempt(ctx, nil)
}

// onConnectFailed is called when an agent was unable to connect to a server.
func (agent *Agent) onConnectFailed(ctx context.Context, err error) {
	agent.logger.Error(err, "failed to connect to the server")
	agent.reporter().RecordConnectionAttempt(ctx, err)
}

// onError is called when an agent receives an error response from the server.
func (agent *Agent) onError(_ context.Context, err *protobufs.ServerErrorResponse) {
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

	if err = agent.rebuildAppliedKeys(); err != nil {
		return fmt.Errorf("failed to rebuild applied keys from cluster state: %w", err)
	}

	settings := types.StartSettings{
		OpAMPServerURL: agent.config.Endpoint,
		Header:         agent.config.Headers.ToHTTPHeader(),
		InstanceUid:    types.InstanceUid(agent.instanceId),
		Callbacks: types.Callbacks{
			OnConnect:              agent.onConnect,
			OnConnectFailed:        agent.onConnectFailed,
			OnError:                agent.onError,
			SaveRemoteConfigStatus: agent.saveRemoteConfigStatus,
			GetEffectiveConfig:     agent.getEffectiveConfig,
			OnMessage:              agent.onMessage,
			OnCommand:              agent.onCommand,
		},
		RemoteConfigStatus:    agent.remoteConfigStatus,
		PackagesStateProvider: nil,
	}

	if agent.config.Proxy != nil && agent.config.Proxy.URL != "" {
		settings.ProxyURL = agent.config.Proxy.URL
		settings.ProxyHeaders = agent.config.Proxy.Headers.ToHTTPHeader()
	}

	// Configure TLS based on explicit tls settings.
	if agent.config.TLS != nil {
		if agent.config.TLS.Insecure {
			parsedURL, parseErr := url.Parse(settings.OpAMPServerURL)
			if parseErr != nil {
				return fmt.Errorf("invalid OpAMP server endpoint %q: %w", settings.OpAMPServerURL, parseErr)
			}

			if parsedURL.Scheme == "wss" || parsedURL.Scheme == "https" {
				return fmt.Errorf(
					"tls.insecure=true requires an insecure endpoint scheme (ws:// or http://), but got %q. "+
						"Please update the endpoint to use the insecure scheme when tls.insecure is enabled",
					settings.OpAMPServerURL,
				)
			}

			agent.logger.Info("TLS is disabled for the OpAMP client connection (tls.insecure=true). This connection is not encrypted.")
		} else if agent.config.TLS.InsecureSkipVerify {
			// TLS enabled but skip certificate verification (standard behavior)
			settings.TLSConfig = &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly opted in via tls.insecure_skip_verify.
			}
			agent.logger.Info("TLS is enabled for the OpAMP client connection but certificate verification is skipped (tls.insecure_skip_verify=true)")
		}
	}

	err = agent.opampClient.SetAgentDescription(agent.agentDescription)
	if err != nil {
		return err
	}
	err = agent.opampClient.SetHealth(agent.getHealth())
	if err != nil {
		return err
	}
	capabilities := agent.config.GetCapabilities()
	err = agent.opampClient.SetCapabilities(&capabilities)
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
	go agent.checkForProxyUpdates()

	agent.logger.V(3).Info("OpAMP Client started.")

	return nil
}

func (agent *Agent) checkForProxyUpdates() {
	for {
		select {
		case <-agent.proxy.HasUpdates():
			agent.logger.Info("new agent connected to proxy")
			err := agent.opampClient.SetHealth(agent.getHealth())
			if err != nil {
				agent.logger.Error(err, "failed to update from proxy")
				return
			}
			updateErr := agent.opampClient.UpdateEffectiveConfig(context.Background())
			if updateErr != nil {
				agent.logger.Error(updateErr, "failed to update from proxy")
				return
			}
		case <-agent.done:
			agent.logger.Info("stopping check for proxy updates")
			return
		}
	}
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
			agent.reporter().RecordHeartbeat(context.Background(), err)
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

// UpdateHealth sends the current health state to the OpAMP server.
func (agent *Agent) UpdateHealth() error {
	return agent.opampClient.SetHealth(agent.getHealth())
}

// updateAgentIdentity receives a new instancedId from the remote server and updates the agent's instanceID field.
// The meter will be reinitialized by the onMessage function.
func (agent *Agent) updateAgentIdentity(instanceId uuid.UUID) {
	agent.logger.V(3).Info("Agent identity is being changed",
		"old instanceId", agent.instanceId.String(),
		"new instanceid", instanceId.String())
	agent.instanceId = instanceId
}

// getEffectiveConfig is called when a remote server needs to learn of the current effective configuration of each
// collector the agent is managing.
func (agent *Agent) getEffectiveConfig(context.Context) (*protobufs.EffectiveConfig, error) {
	instances, err := agent.applier.ListInstances()
	if err != nil {
		agent.logger.Error(err, "failed to list instances")
		return nil, err
	}
	instanceMap := map[string]*protobufs.AgentConfigObject{}
	for _, instance := range instances {
		if instance.GetDeletionTimestamp() != nil {
			continue
		}
		for key, file := range instance.GetConfigMap() {
			if file.Body == nil {
				agent.logger.Error(errors.New("nil effective config"), "failed to get effective config",
					"name", instance.GetName(), "namespace", instance.GetNamespace(), "key", key)
				continue
			}
			contentType := file.ContentType
			if contentType == "" {
				contentType = "yaml"
			}
			instanceMap[key] = &protobufs.AgentConfigObject{
				Body:        file.Body,
				ContentType: contentType,
			}
		}
	}
	for id, instance := range agent.proxy.GetConfigurations() {
		if cfg, ok := instance.GetConfigMap().GetConfigMap()[""]; ok {
			instanceMap[id.String()] = cfg
		}
	}
	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: instanceMap,
		},
	}, nil
}

// initMeter configures the bridge's own metrics from the TelemetryConnectionSettings offered by the OpAMP server.
//
//   - Settings identical to the ones currently in use are ignored, so a server that repeats its offer on every
//     message does not reset the bridge's counters.
//   - Settings with an empty destination endpoint stop own-metrics reporting.
//   - Otherwise a new reporter is created and replaces (and shuts down) the previous one. If the new settings are
//     invalid, the previous reporter is left running.
//
// forceRecreate rebuilds the reporter even when the settings are unchanged, for example because the instance ID
// reported in the metric resource has changed.
func (agent *Agent) initMeter(settings *protobufs.TelemetryConnectionSettings, forceRecreate bool) {
	if !forceRecreate && agent.ownMetricsSettings != nil && proto.Equal(agent.ownMetricsSettings, settings) {
		agent.logger.V(4).Info("own metrics connection settings unchanged")
		return
	}

	if settings.GetDestinationEndpoint() == "" {
		agent.logger.Info("own metrics destination removed by the server, stopping metrics reporting")
		agent.ownMetricsSettings = nil
		if previous := agent.metricReporter.Swap(nil); previous != nil {
			previous.Shutdown()
		}
		return
	}

	reporter, err := agent.newReporter(agent.logger.WithName("metrics"), settings, agent.config.GetAgentType(), agent.config.GetAgentVersion(), agent.instanceId)
	if err != nil {
		agent.logger.Error(err, "failed to create metric reporter")
		return
	}

	agent.ownMetricsSettings = proto.Clone(settings).(*protobufs.TelemetryConnectionSettings)
	if previous := agent.metricReporter.Swap(reporter); previous != nil {
		previous.Shutdown()
	}
	agent.logger.Info("reporting own metrics", "endpoint", settings.GetDestinationEndpoint())
}

// rebuildAppliedKeys reconstructs appliedKeys from the collectors currently in the cluster. appliedKeys
// only lives in memory, so without this a restart forgets every key the bridge previously applied and a
// later remote config that drops one of those keys never triggers the corresponding delete.
func (agent *Agent) rebuildAppliedKeys() error {
	instances, err := agent.applier.ListInstances()
	if err != nil {
		return err
	}
	for _, instance := range instances {
		if !instance.IsManaged() || instance.GetDeletionTimestamp() != nil {
			continue
		}
		for key := range instance.GetConfigMap() {
			agent.appliedKeys[key] = true
		}
	}
	return nil
}

// applyRemoteConfig receives a remote configuration from a remote server of the following form:
//
//	map[resource key] -> AgentConfigObject body
//
// For every key in the received remote configuration, the agent attempts to apply it via the configured
// applier. If an entry fails to apply, the agent continues to the next entry. The agent stores the
// received configuration hash regardless of application status, as per the OpAMP spec.
//
// INVARIANT: The caller must verify that config isn't nil _and_ the configuration has changed between calls.
func (agent *Agent) applyRemoteConfig(ctx context.Context, config *protobufs.AgentRemoteConfig) (*protobufs.RemoteConfigStatus, error) {
	reporter := agent.reporter()
	var errs []error
	// Apply changes from the received config map
	for key, file := range config.Config.GetConfigMap() {
		if key == "" {
			err := errors.New("remote config entry has empty name")
			reporter.RecordConfigOperation(ctx, metrics.OperationApply, err)
			errs = append(errs, err)
			continue
		}
		if len(file.Body) == 0 {
			err := fmt.Errorf("remote config entry %q has empty body", key)
			reporter.RecordConfigOperation(ctx, metrics.OperationApply, err)
			errs = append(errs, err)
			continue
		}
		err := agent.applier.Apply(key, file)
		reporter.RecordConfigOperation(ctx, metrics.OperationApply, err)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		agent.appliedKeys[key] = true
	}
	// Check if anything was deleted
	for key := range agent.appliedKeys {
		if _, ok := config.Config.GetConfigMap()[key]; !ok {
			err := agent.applier.Delete(key)
			reporter.RecordConfigOperation(ctx, metrics.OperationDelete, err)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	agent.lastHash = config.GetConfigHash()
	multiErr := errors.Join(errs...)
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
	if reporter := agent.metricReporter.Swap(nil); reporter != nil {
		reporter.Shutdown()
	}
}

// onCommand is called when the OpAMP server sends a ServerToAgentCommand.
// Only CommandType_Restart is handled; all other command types return an error to the server.
func (agent *Agent) onCommand(ctx context.Context, command *protobufs.ServerToAgentCommand) error {
	err := agent.handleCommand(ctx, command)
	agent.reporter().RecordCommand(ctx, command.GetType(), err)
	return err
}

func (agent *Agent) handleCommand(ctx context.Context, command *protobufs.ServerToAgentCommand) error {
	switch command.GetType() {
	case protobufs.CommandType_CommandType_Restart:
		if !agent.config.RestartCommandEnabled() {
			err := errors.New("restart command rejected: AcceptsRestartCommand capability is not enabled")
			agent.logger.Error(err, "Received restart command without capability")
			return err
		}
		agent.logger.Info("Received restart command, triggering collector restart")
		if err := agent.applier.Restart(ctx); err != nil {
			agent.logger.Error(err, "Failed to restart collector on command")
			return err
		}
		return nil
	default:
		err := fmt.Errorf("unsupported command type: %s", command.GetType().String())
		agent.logger.Error(err, "Received unsupported command type")
		return err
	}
}

// onMessage is called when the client receives a new message from the connected OpAMP server. The agent is responsible
// for checking if it should apply a new remote configuration. The agent will also initialize metrics based on the
// settings received from the server. The agent is also able to update its identifier if it needs to.
func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {
	// The instance id is updated prior to the meter initialization so that the new meter will report using the updated
	// instanceId.
	identityChanged := false
	if msg.AgentIdentification != nil {
		uid, err := uuid.FromBytes(msg.AgentIdentification.NewInstanceUid)
		if err != nil {
			agent.logger.Error(err, "couldn't parse instance UID")
		} else {
			identityChanged = uid != agent.instanceId
			agent.updateAgentIdentity(uid)
		}
	}

	// Own metrics are configured before the remote configuration is applied, so that the work done for a remote
	// configuration received in the same message is already reported.
	switch {
	case msg.OwnMetricsConnSettings != nil:
		agent.initMeter(msg.OwnMetricsConnSettings, identityChanged)
	case identityChanged && agent.ownMetricsSettings != nil:
		// Rebuild the reporter so its resource carries the new service.instance.id.
		agent.initMeter(agent.ownMetricsSettings, true)
	}

	// If we received remote configuration, and it's not the same as the previously applied one
	if agent.remoteConfigEnabled && msg.RemoteConfig != nil && !bytes.Equal(agent.lastHash, msg.RemoteConfig.GetConfigHash()) {
		start := time.Now()
		status, err := agent.applyRemoteConfig(ctx, msg.RemoteConfig)
		agent.reporter().RecordRemoteConfig(ctx, time.Since(start), err)
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
