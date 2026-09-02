// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// NoopServer satisfies the agent's proxy dependency when no OpAMP proxy is running.
type NoopServer struct{}

func (NoopServer) GetAgentsByHostname() map[string]uuid.UUID {
	return map[string]uuid.UUID{}
}

func (NoopServer) GetConfigurations() map[uuid.UUID]*protobufs.EffectiveConfig {
	return map[uuid.UUID]*protobufs.EffectiveConfig{}
}

func (NoopServer) GetHealth() map[uuid.UUID]*protobufs.ComponentHealth {
	return map[uuid.UUID]*protobufs.ComponentHealth{}
}

func (NoopServer) HasUpdates() <-chan struct{} {
	return make(chan struct{})
}
