// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"log"
	"maps"
	"sync"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
)

var logger = log.New(log.Default().Writer(), "[AGENTS] ", log.Default().Flags()|log.Lmsgprefix|log.Lmicroseconds)

type Agents struct {
	mux         sync.RWMutex
	agentsById  map[InstanceId]*Agent
	connections map[types.Connection]map[InstanceId]bool
}

func NewAgents() *Agents {
	return &Agents{
		agentsById:  map[InstanceId]*Agent{},
		connections: map[types.Connection]map[InstanceId]bool{},
	}
}

// RemoveConnection removes the connection all Agent instances associated with the
// connection.
func (a *Agents) RemoveConnection(conn types.Connection) {
	a.mux.Lock()
	defer a.mux.Unlock()

	for instanceId := range a.connections[conn] {
		delete(a.agentsById, instanceId)
	}
	delete(a.connections, conn)
}

func (a *Agents) SetCustomConfigForAgent(
	agentId InstanceId,
	config *protobufs.AgentConfigMap,
	notifyNextStatusUpdate chan<- struct{},
) {
	agent := a.FindAgent(agentId)
	if agent != nil {
		agent.SetCustomConfig(config, notifyNextStatusUpdate)
	}
}

// SendCommandToAgent sends a ServerToAgentCommand to the specified agent.
func (a *Agents) SendCommandToAgent(agentId InstanceId, commandType protobufs.CommandType) {
	agent := a.FindAgent(agentId)
	if agent != nil {
		agent.SendToAgent(&protobufs.ServerToAgent{
			Command: &protobufs.ServerToAgentCommand{Type: commandType},
		})
	}
}

func (a *Agents) FindAgent(agentId InstanceId) *Agent {
	a.mux.RLock()
	defer a.mux.RUnlock()
	return a.agentsById[agentId]
}

func (a *Agents) FindOrCreateAgent(agentId InstanceId, conn types.Connection) *Agent {
	a.mux.Lock()
	defer a.mux.Unlock()

	// Ensure the Agent is in the agentsById map.
	agent := a.agentsById[agentId]
	if agent == nil {
		agent = NewAgent(agentId, conn)
		a.agentsById[agentId] = agent

		// Ensure the Agent's instance id is associated with the connection.
		if a.connections[conn] == nil {
			a.connections[conn] = map[InstanceId]bool{}
		}
		a.connections[conn][agentId] = true
	}

	return agent
}

func (a *Agents) GetAgentReadonlyClone(agentId InstanceId) *Agent {
	agent := a.FindAgent(agentId)
	if agent == nil {
		return nil
	}

	// Return a clone to allow safe access after returning.
	return agent.CloneReadonly()
}

func (a *Agents) GetAllAgentsReadonlyClone() map[InstanceId]*Agent {
	a.mux.RLock()

	// Clone the map first
	m := map[InstanceId]*Agent{}
	maps.Copy(m, a.agentsById)
	a.mux.RUnlock()

	// Clone agents in the map
	for id, agent := range m {
		// Return a clone to allow safe access after returning.
		m[id] = agent.CloneReadonly()
	}
	return m
}

func (a *Agents) OfferAgentConnectionSettings(
	id InstanceId,
	offers *protobufs.ConnectionSettingsOffers,
) {
	logger.Printf("Begin rotate client certificate for %s\n", id)

	a.mux.Lock()
	defer a.mux.Unlock()

	agent, ok := a.agentsById[id]
	if ok {
		agent.OfferConnectionSettings(offers)
		logger.Printf("Client certificate offers sent to %s\n", id)
	} else {
		logger.Printf("Agent %s not found\n", id)
	}
}
