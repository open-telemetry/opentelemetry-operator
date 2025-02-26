package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"google.golang.org/protobuf/proto"
)

var _ json.Marshaler = &Agent{}

type InstanceId uuid.UUID

// Agent represents a connected Agent.
type Agent struct {
	// Some fields in this struct are exported so that we can render them in the UI.

	// Agent's instance id. This is an immutable field.
	InstanceId    InstanceId
	InstanceIdStr string

	// Connection to the Agent.
	conn types.Connection

	// mutex for the fields that follow it.
	mux sync.RWMutex

	// Agent's current status.
	Status *protobufs.AgentToServer

	// The time when the agent has started. Valid only if Status.Health.Up==true
	StartedAt time.Time

	// Effective config reported by the Agent.
	EffectiveConfig map[string]string

	// Optional special remote config for this particular instance defined by
	// the user in the UI.
	CustomInstanceConfig map[string]string

	// Remote config that we will give to this Agent.
	remoteConfig *protobufs.AgentRemoteConfig

	// Channels to notify when this Agent's status is updated next time.
	statusUpdateWatchers []chan<- struct{}
}

func (agent *Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Status          *protobufs.AgentToServer `json:"status"`
		StartedAt       time.Time                `json:"started_at"`
		EffectiveConfig map[string]string        `json:"effective_config"`
	}{
		Status:          agent.Status,
		StartedAt:       agent.StartedAt,
		EffectiveConfig: agent.EffectiveConfig,
	})
}

func NewAgent(
	instanceId InstanceId,
	conn types.Connection,
) *Agent {
	agent := &Agent{InstanceId: instanceId, InstanceIdStr: uuid.UUID(instanceId).String(), conn: conn}

	return agent
}

// CloneReadonly returns a copy of the Agent that is safe to read.
// Functions that modify the Agent should not be called on the cloned copy.
func (agent *Agent) CloneReadonly() *Agent {
	agent.mux.RLock()
	defer agent.mux.RUnlock()
	return &Agent{
		InstanceId:           agent.InstanceId,
		InstanceIdStr:        uuid.UUID(agent.InstanceId).String(),
		Status:               proto.Clone(agent.Status).(*protobufs.AgentToServer),
		EffectiveConfig:      agent.EffectiveConfig,
		CustomInstanceConfig: agent.CustomInstanceConfig,
		remoteConfig:         proto.Clone(agent.remoteConfig).(*protobufs.AgentRemoteConfig),
		StartedAt:            agent.StartedAt,
	}
}

// UpdateStatus updates the status of the Agent struct based on the newly received
// status report and sets appropriate fields in the response message to be sent
// to the Agent.
func (agent *Agent) UpdateStatus(
	statusMsg *protobufs.AgentToServer,
	response *protobufs.ServerToAgent,
) {
	agent.mux.Lock()

	agent.processStatusUpdate(statusMsg, response)

	if statusMsg.ConnectionSettingsRequest != nil {
		//agent.processConnectionSettingsRequest(statusMsg.ConnectionSettingsRequest.Opamp, response)
	}

	statusUpdateWatchers := agent.statusUpdateWatchers
	agent.statusUpdateWatchers = nil

	agent.mux.Unlock()

	// Notify watcher outside mutex to avoid blocking the mutex for too long.
	notifyStatusWatchers(statusUpdateWatchers)
}

func notifyStatusWatchers(statusUpdateWatchers []chan<- struct{}) {
	// Notify everyone who is waiting on this Agent's status updates.
	for _, ch := range statusUpdateWatchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (agent *Agent) updateAgentDescription(newStatus *protobufs.AgentToServer) (agentDescrChanged bool) {
	prevStatus := agent.Status

	if agent.Status == nil {
		// First time this Agent reports a status, remember it.
		agent.Status = newStatus
		agentDescrChanged = true
	} else {
		// Not a new Agent. Update the Status.
		agent.Status.SequenceNum = newStatus.SequenceNum

		// Check what's changed in the AgentDescription.
		if newStatus.AgentDescription != nil {
			// If the AgentDescription field is set it means the Agent tells us
			// something is changed in the field since the last status report
			// (or this is the first report).
			// Make full comparison of previous and new descriptions to see if it
			// really is different.
			if prevStatus != nil && proto.Equal(prevStatus.AgentDescription, newStatus.AgentDescription) {
				// Agent description didn't change.
				agentDescrChanged = false
			} else {
				// Yes, the description is different, update it.
				agent.Status.AgentDescription = newStatus.AgentDescription
				agentDescrChanged = true
			}
		} else {
			// AgentDescription field is not set, which means description didn't change.
			agentDescrChanged = false
		}

		// Update remote config status if it is included and is different from what we have.
		if newStatus.RemoteConfigStatus != nil &&
			!proto.Equal(agent.Status.RemoteConfigStatus, newStatus.RemoteConfigStatus) {
			agent.Status.RemoteConfigStatus = newStatus.RemoteConfigStatus
		}
	}
	return agentDescrChanged
}

func (agent *Agent) updateHealth(newStatus *protobufs.AgentToServer) {
	if newStatus.Health == nil {
		return
	}

	agent.Status.Health = newStatus.Health

	if agent.Status != nil && agent.Status.Health != nil && agent.Status.Health.Healthy {
		agent.StartedAt = time.Unix(0, int64(agent.Status.Health.StartTimeUnixNano)).UTC()
	}
}

func (agent *Agent) updateRemoteConfigStatus(newStatus *protobufs.AgentToServer) {
	// Update remote config status if it is included and is different from what we have.
	if newStatus.RemoteConfigStatus != nil {
		agent.Status.RemoteConfigStatus = newStatus.RemoteConfigStatus
	}
}

func (agent *Agent) updateStatusField(newStatus *protobufs.AgentToServer) (agentDescrChanged bool) {
	if agent.Status == nil {
		// First time this Agent reports a status, remember it.
		agent.Status = newStatus
		agentDescrChanged = true
	}

	agentDescrChanged = agent.updateAgentDescription(newStatus) || agentDescrChanged
	agent.updateRemoteConfigStatus(newStatus)
	agent.updateHealth(newStatus)

	return agentDescrChanged
}

func (agent *Agent) updateEffectiveConfig(
	newStatus *protobufs.AgentToServer,
	response *protobufs.ServerToAgent,
) {
	// Update effective config if provided.
	if newStatus.EffectiveConfig != nil {
		if newStatus.EffectiveConfig.ConfigMap != nil {
			agent.Status.EffectiveConfig = newStatus.EffectiveConfig

			// Convert to string for displaying purposes.
			agent.EffectiveConfig = map[string]string{}
			for key, cfg := range newStatus.EffectiveConfig.ConfigMap.ConfigMap {
				// TODO: we just concatenate parts of effective config as a single
				// blob to show in the UI. A proper approach is to keep the effective
				// config as a set and show the set in the UI.
				agent.EffectiveConfig[key] = string(cfg.Body)
			}
		}
	}
}

func (agent *Agent) hasCapability(capability protobufs.AgentCapabilities) bool {
	return agent.Status.Capabilities&uint64(capability) != 0
}

func (agent *Agent) processStatusUpdate(
	newStatus *protobufs.AgentToServer,
	response *protobufs.ServerToAgent,
) {
	// We don't have any status for this Agent, or we lost the previous status update from the Agent, so our
	// current status is not up-to-date.
	lostPreviousUpdate := (agent.Status == nil) || (agent.Status != nil && agent.Status.SequenceNum+1 != newStatus.SequenceNum)

	agentDescrChanged := agent.updateStatusField(newStatus)

	// Check if any fields were omitted in the status report.
	effectiveConfigOmitted := newStatus.EffectiveConfig == nil &&
		agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig)

	packageStatusesOmitted := newStatus.PackageStatuses == nil &&
		agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsPackageStatuses)

	remoteConfigStatusOmitted := newStatus.RemoteConfigStatus == nil &&
		agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig)

	healthOmitted := newStatus.Health == nil &&
		agent.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth)

	// True if the status was not fully reported.
	statusIsCompressed := effectiveConfigOmitted || packageStatusesOmitted || remoteConfigStatusOmitted || healthOmitted

	if statusIsCompressed && lostPreviousUpdate {
		// The status message is not fully set in the message that we received, but we lost the previous
		// status update. Request full status update from the agent.
		response.Flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	configChanged := false
	if agentDescrChanged {
		// Agent description is changed.

		// We need to recalculate the config.
		configChanged = agent.calcRemoteConfig()

		// And set connection settings that are appropriate for the Agent description.
		agent.calcConnectionSettings(response)
	}

	// If remote config is changed and different from what the Agent has then
	// send the new remote config to the Agent.
	if configChanged ||
		(agent.Status.RemoteConfigStatus != nil &&
			bytes.Compare(agent.Status.RemoteConfigStatus.LastRemoteConfigHash, agent.remoteConfig.ConfigHash) != 0) {
		// The new status resulted in a change in the config of the Agent or the Agent
		// does not have this config (hash is different). Send the new config the Agent.
		response.RemoteConfig = agent.remoteConfig
	}

	agent.updateEffectiveConfig(newStatus, response)
}

// SetCustomConfig sets a custom config for this Agent.
// notifyWhenConfigIsApplied channel is notified after the remote config is applied
// to the Agent and after the Agent reports back the effective config.
// If the provided config is equal to the current remoteConfig of the Agent
// then we will not send any config to the Agent and notifyWhenConfigIsApplied channel
// will be notified immediately. This requires that notifyWhenConfigIsApplied channel
// has a buffer size of at least 1.
func (agent *Agent) SetCustomConfig(
	config *protobufs.AgentConfigMap,
	notifyWhenConfigIsApplied chan<- struct{},
) {
	agent.mux.Lock()

	for key, file := range config.GetConfigMap() {
		agent.CustomInstanceConfig[key] = string(file.Body)
		agent.EffectiveConfig[key] = string(file.Body)
	}

	configChanged := agent.calcRemoteConfig()
	if configChanged {
		if notifyWhenConfigIsApplied != nil {
			// The caller wants to be notified when the Agent reports a status
			// update next time. This is typically used in the UI to wait until
			// the configuration changes are propagated successfully to the Agent.
			agent.statusUpdateWatchers = append(
				agent.statusUpdateWatchers,
				notifyWhenConfigIsApplied,
			)
		}
		msg := &protobufs.ServerToAgent{
			RemoteConfig: agent.remoteConfig,
		}
		agent.mux.Unlock()

		agent.SendToAgent(msg)
	} else {
		agent.mux.Unlock()

		if notifyWhenConfigIsApplied != nil {
			// No config change. We are not going to send config to the Agent and
			// as a result we do not expect status update from the Agent, so we will
			// just notify the waiter that the config change is done.
			notifyWhenConfigIsApplied <- struct{}{}
		}
	}
}

// calcRemoteConfig calculates the remote config for this Agent. It returns true if
// the calculated new config is different from the existing config stored in
// Agent.remoteConfig.
func (agent *Agent) calcRemoteConfig() bool {
	hash := sha256.New()

	cfg := protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{},
		},
	}

	// Add the custom config for this particular Agent instance. Use empty
	// string as the config file name.
	for key, body := range agent.CustomInstanceConfig {
		cfg.Config.ConfigMap[key] = &protobufs.AgentConfigFile{
			Body: []byte(body),
		}
	}

	// Calculate the hash.
	for k, v := range cfg.Config.ConfigMap {
		hash.Write([]byte(k))
		hash.Write(v.Body)
		hash.Write([]byte(v.ContentType))
	}

	cfg.ConfigHash = hash.Sum(nil)

	configChanged := !isEqualRemoteConfig(agent.remoteConfig, &cfg)

	agent.remoteConfig = &cfg

	return configChanged
}

func isEqualRemoteConfig(c1, c2 *protobufs.AgentRemoteConfig) bool {
	if c1 == c2 {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return isEqualConfigSet(c1.Config, c2.Config)
}

func isEqualConfigSet(c1, c2 *protobufs.AgentConfigMap) bool {
	if c1 == c2 {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	if len(c1.ConfigMap) != len(c2.ConfigMap) {
		return false
	}
	for k, v1 := range c1.ConfigMap {
		v2, ok := c2.ConfigMap[k]
		if !ok {
			return false
		}
		if !isEqualConfigFile(v1, v2) {
			return false
		}
	}
	return true
}

func isEqualConfigFile(f1, f2 *protobufs.AgentConfigFile) bool {
	if f1 == f2 {
		return true
	}
	if f1 == nil || f2 == nil {
		return false
	}
	return bytes.Compare(f1.Body, f2.Body) == 0 && f1.ContentType == f2.ContentType
}

func (agent *Agent) calcConnectionSettings(response *protobufs.ServerToAgent) {
	// Here we can use Agent's description to send the appropriate connection
	// settings to the Agent.
	// In this simple example the connection settings do not depend on the
	// Agent description, so we jst set them directly.

	response.ConnectionSettings = &protobufs.ConnectionSettingsOffers{
		Hash:       nil, // TODO: calc has from settings.
		Opamp:      nil,
		OwnMetrics: nil,
		//&protobufs.TelemetryConnectionSettings{
		//	// We just hard-code this to a port on a localhost on which we can
		//	// run an Otel Collector for demo purposes. With real production
		//	// servers this should likely point to an OTLP backend.
		//	DestinationEndpoint: "http://localhost:4318/v1/metrics",
		//},
		OwnTraces:        nil,
		OwnLogs:          nil,
		OtherConnections: nil,
	}
}

func (agent *Agent) SendToAgent(msg *protobufs.ServerToAgent) {
	agent.conn.Send(context.Background(), msg)
}

func (agent *Agent) OfferConnectionSettings(offers *protobufs.ConnectionSettingsOffers) {
	agent.SendToAgent(
		&protobufs.ServerToAgent{
			ConnectionSettings: offers,
		},
	)
}

func (agent *Agent) addErrorResponse(errMsg string, response *protobufs.ServerToAgent) {
	logger.Println(errMsg)
	if response.ErrorResponse == nil {
		response.ErrorResponse = &protobufs.ServerErrorResponse{
			Type:         protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest,
			ErrorMessage: errMsg,
			Details:      nil,
		}
	} else if response.ErrorResponse.Type == protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest {
		// Append this error message to the existing error message.
		response.ErrorResponse.ErrorMessage += errMsg
	} else {
		// Can't report it since it is a different error type.
		// TODO: consider adding support for reporting multiple errors of different type in the response.
	}
}
