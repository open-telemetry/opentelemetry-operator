// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

func TestManager_StartStartsRuntimesBeforeKubernetesClient(t *testing.T) {
	cfg := config.NewConfig(logr.Discard())
	cfg.Endpoint = "ws://example.test/v1/opamp"
	cfg.Capabilities = map[config.Capability]bool{
		config.ReportsHealth: true,
	}
	opampClient := &recordingOpAMPClient{}
	agent := opampagent.NewAgent(logr.Discard(), healthyApplier{}, cfg, opampClient, proxy.NoopServer{})
	kubernetesClient := &orderCheckingKubernetesClient{opampClient: opampClient}
	manager, err := New(
		WithLogger(logr.Discard()),
		WithRuntimes([]Runtime{{Name: "test", OpAMPAgent: agent, Client: opampClient}}),
		WithKubernetesClient(kubernetesClient),
	)
	require.NoError(t, err)

	require.NoError(t, manager.Start(context.Background()))
	t.Cleanup(func() {
		manager.Shutdown(t.Context())
	})
	require.True(t, kubernetesClient.started)
}

func TestManager_NewRequiresPermissionReviewClientWithRequiredPermissions(t *testing.T) {
	manager, err := New(
		WithRequiredPermissions(func() ([]Permission, error) {
			return []Permission{{Verb: "get", Resource: "pods"}}, nil
		}),
	)

	require.Nil(t, manager)
	require.EqualError(t, err, "permission review client is required")
}

func TestManager_NewRequiresRuntimeOpAMPAgent(t *testing.T) {
	manager, err := New(
		WithRuntimes([]Runtime{{Name: "test"}}),
	)

	require.Nil(t, manager)
	require.EqualError(t, err, "runtime 0 opamp agent is required")
}

type orderCheckingKubernetesClient struct {
	opampClient *recordingOpAMPClient
	started     bool
}

func (c *orderCheckingKubernetesClient) Start(context.Context) error {
	if !c.opampClient.Started() {
		return errors.New("kubernetes client started before opamp client")
	}
	c.started = true
	return nil
}

type healthyApplier struct{}

func (healthyApplier) Apply(string, *protobufs.AgentConfigFile) error {
	return nil
}

func (healthyApplier) Delete(string) error {
	return nil
}

func (healthyApplier) ListInstances() ([]operator.CollectorInstance, error) {
	return nil, nil
}

func (healthyApplier) GetHealth() (operator.Health, error) {
	return operator.Health{Healthy: true, Children: map[string]operator.Health{}}, nil
}

type recordingOpAMPClient struct {
	mu      sync.Mutex
	started bool
}

func (c *recordingOpAMPClient) Started() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.started
}

func (c *recordingOpAMPClient) Start(context.Context, types.StartSettings) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = true
	return nil
}

func (c *recordingOpAMPClient) Stop(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = false
	return nil
}

func (*recordingOpAMPClient) SetAgentDescription(*protobufs.AgentDescription) error {
	return nil
}

func (*recordingOpAMPClient) AgentDescription() *protobufs.AgentDescription {
	return nil
}

func (*recordingOpAMPClient) SetHealth(*protobufs.ComponentHealth) error {
	return nil
}

func (*recordingOpAMPClient) UpdateEffectiveConfig(context.Context) error {
	return nil
}

func (*recordingOpAMPClient) SetRemoteConfigStatus(*protobufs.RemoteConfigStatus) error {
	return nil
}

func (*recordingOpAMPClient) SetPackageStatuses(*protobufs.PackageStatuses) error {
	return nil
}

func (*recordingOpAMPClient) RequestConnectionSettings(*protobufs.ConnectionSettingsRequest) error {
	return nil
}

func (*recordingOpAMPClient) SetCustomCapabilities(*protobufs.CustomCapabilities) error {
	return nil
}

func (*recordingOpAMPClient) SetFlags(protobufs.AgentToServerFlags) {}

func (*recordingOpAMPClient) SendCustomMessage(*protobufs.CustomMessage) (chan struct{}, error) {
	return nil, nil
}

func (*recordingOpAMPClient) SetAvailableComponents(*protobufs.AvailableComponents) error {
	return nil
}

func (*recordingOpAMPClient) SetCapabilities(*protobufs.AgentCapabilities) error {
	return nil
}
