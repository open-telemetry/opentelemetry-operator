// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"google.golang.org/protobuf/proto"
)

type Agent struct {
	// logger for the agentImpl
	logger logr.Logger
	// Agent's instance id. This is an immutable field.
	InstanceId uuid.UUID
	// Connection to the Agent.
	conn types.Connection
	// mutex for the fields that follow it.
	mux sync.RWMutex

	health          *protobufs.ComponentHealth
	effectiveConfig *protobufs.EffectiveConfig
	// Agent's current status.
	Status *protobufs.AgentToServer
}

func NewAgent(logger logr.Logger, agentId uuid.UUID, conn types.Connection) *Agent {
	return &Agent{
		logger:     logger,
		InstanceId: agentId,
		conn:       conn,
	}
}

func (a *Agent) hasCapability(capability protobufs.AgentCapabilities) bool {
	if capability < 0 {
		return false
	}
	//nolint:gosec
	return a.Status.Capabilities&uint64(capability) != 0
}

// updateAgentDescription assumes that the status is already non-nil.
func (a *Agent) updateAgentDescription(newStatus *protobufs.AgentToServer) (agentDescrChanged bool) {
	prevStatus := a.Status

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
			a.Status.AgentDescription = newStatus.AgentDescription
			agentDescrChanged = true
		}
	} else {
		// AgentDescription field is not set, which means description didn't change.
		agentDescrChanged = false
	}
	return agentDescrChanged
}

func (a *Agent) updateStatusField(newStatus *protobufs.AgentToServer) (agentDescrChanged bool) {
	if a.Status == nil {
		// First time this Agent reports a status, remember it.
		a.Status = newStatus
		agentDescrChanged = true
	}
	a.Status.SequenceNum = newStatus.SequenceNum
	return agentDescrChanged || a.updateAgentDescription(newStatus)
}

func (a *Agent) UpdateStatus(newStatus *protobufs.AgentToServer, response *protobufs.ServerToAgent) bool {
	a.mux.Lock()
	defer a.mux.Unlock()
	// We don't have any status for this Agent, or we lost the previous status update from the Agent, so our
	// current status is not up-to-date.
	lostPreviousUpdate := (a.Status == nil) || (a.Status != nil && a.Status.SequenceNum+1 != newStatus.SequenceNum)

	agentDescrChanged := a.updateStatusField(newStatus)
	// Check if any fields were omitted in the status report.
	effectiveConfigOmitted := newStatus.EffectiveConfig == nil &&
		a.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig)

	packageStatusesOmitted := newStatus.PackageStatuses == nil &&
		a.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsPackageStatuses)

	remoteConfigStatusOmitted := newStatus.RemoteConfigStatus == nil &&
		a.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig)

	healthOmitted := newStatus.Health == nil &&
		a.hasCapability(protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth)

	// True if the status was not fully reported.
	statusIsCompressed := effectiveConfigOmitted || packageStatusesOmitted || remoteConfigStatusOmitted || healthOmitted
	if statusIsCompressed && lostPreviousUpdate {
		// The status message is not fully set in the message that we received, but we lost the previous
		// status update. Request full status update from the a.
		response.Flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	if agentDescrChanged {
		// Agent description is changed.
		// And set connection settings that are appropriate for the Agent description.
		a.calcConnectionSettings(response)
	}
	if newStatus.Health != nil {
		a.health = newStatus.Health
	}
	if newStatus.EffectiveConfig != nil {
		a.effectiveConfig = newStatus.EffectiveConfig
	}
	if newStatus.CustomMessage != nil {
		a.logger.V(5).Info("received custom message, not implemented")
	}
	return agentDescrChanged
}

func (a *Agent) calcConnectionSettings(response *protobufs.ServerToAgent) {
	// Here we can use Agent's description to send the appropriate connection
	// settings to the Agent.
	// In this simple example the connection settings do not depend on the
	// Agent description, so we just set them directly.

	response.ConnectionSettings = &protobufs.ConnectionSettingsOffers{
		Hash:  nil, // TODO: calc has from settings.
		Opamp: nil,
		// TODO: The own settings should be sent to the gateway for leaf nodes automatically.
		OwnMetrics:       nil,
		OwnTraces:        nil,
		OwnLogs:          nil,
		OtherConnections: nil,
	}
}

func (a *Agent) GetHealth() *protobufs.ComponentHealth {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.health
}

func (a *Agent) GetConfiguration() *protobufs.EffectiveConfig {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.effectiveConfig
}

func (a *Agent) GetHostname() string {
	a.mux.RLock()
	defer a.mux.RUnlock()

	hostName := ""
	if a.Status != nil && a.Status.AgentDescription != nil {
		for _, kv := range a.Status.AgentDescription.GetNonIdentifyingAttributes() {
			if kv.Key == string(semconv.HostNameKey) {
				hostName = kv.Value.GetStringValue()
			}
		}
	}
	return hostName
}
