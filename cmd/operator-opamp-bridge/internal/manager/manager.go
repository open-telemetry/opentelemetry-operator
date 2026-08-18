// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	opampclient "github.com/open-telemetry/opamp-go/client"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/healthcheck"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

type Runtime struct {
	Name       string
	OpAMPAgent *opampagent.Agent
	Client     opampclient.OpAMPClient
}

type KubernetesClient interface {
	Start(context.Context) error
}

type Option func(*Manager)

type Manager struct {
	log                     logr.Logger
	runtimes                []Runtime
	healthServer            *healthcheck.Server
	opampProxy              *proxy.OpAMPProxy
	kubernetesClient        KubernetesClient
	permissionReviewClient  PermissionReviewClient
	listRequiredPermissions func() ([]Permission, error)
	cancelKubernetesClient  context.CancelFunc
	shutdownOnce            sync.Once
}

func WithLogger(log logr.Logger) Option {
	return func(m *Manager) {
		m.log = log
	}
}

func WithRuntimes(runtimes []Runtime) Option {
	return func(m *Manager) {
		m.runtimes = runtimes
	}
}

func WithHealthServer(healthServer *healthcheck.Server) Option {
	return func(m *Manager) {
		m.healthServer = healthServer
	}
}

func WithOpAMPProxy(opampProxy *proxy.OpAMPProxy) Option {
	return func(m *Manager) {
		m.opampProxy = opampProxy
	}
}

func WithKubernetesClient(kubernetesClient KubernetesClient) Option {
	return func(m *Manager) {
		m.kubernetesClient = kubernetesClient
	}
}

func WithPermissionReviewClient(permissionReviewClient PermissionReviewClient) Option {
	return func(m *Manager) {
		m.permissionReviewClient = permissionReviewClient
	}
}

func WithRequiredPermissions(listRequiredPermissions func() ([]Permission, error)) Option {
	return func(m *Manager) {
		m.listRequiredPermissions = listRequiredPermissions
	}
}

func New(options ...Option) (*Manager, error) {
	manager := &Manager{}
	for _, option := range options {
		option(manager)
	}
	if err := manager.validate(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) validate() error {
	if m.listRequiredPermissions != nil && m.permissionReviewClient == nil {
		return errors.New("permission review client is required")
	}
	for i, runtime := range m.runtimes {
		if runtime.OpAMPAgent == nil {
			return fmt.Errorf("runtime %d opamp agent is required", i)
		}
	}
	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	if m.listRequiredPermissions != nil {
		requiredPermissions, err := m.listRequiredPermissions()
		if err != nil {
			return err
		}
		if err := CheckPermissions(ctx, m.permissionReviewClient, requiredPermissions); err != nil {
			return err
		}
	}
	for _, runtime := range m.runtimes {
		if err := runtime.OpAMPAgent.Start(); err != nil {
			m.Shutdown(ctx)
			return err
		}
	}
	if m.kubernetesClient != nil {
		kubernetesClientCtx, cancelKubernetesClient := context.WithCancel(ctx)
		m.cancelKubernetesClient = cancelKubernetesClient
		if err := m.kubernetesClient.Start(kubernetesClientCtx); err != nil {
			cancelKubernetesClient()
			m.cancelKubernetesClient = nil
			m.Shutdown(ctx)
			return err
		}
	}
	if m.opampProxy != nil {
		if err := m.opampProxy.Start(); err != nil {
			m.Shutdown(ctx)
			return err
		}
	}
	if m.healthServer != nil {
		if err := m.healthServer.Start(ctx); err != nil {
			m.Shutdown(ctx)
			return err
		}
	}
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) {
	m.shutdownOnce.Do(func() {
		if m.cancelKubernetesClient != nil {
			m.cancelKubernetesClient()
		}
		for _, runtime := range m.runtimes {
			runtime.OpAMPAgent.Shutdown()
		}
		if m.opampProxy != nil {
			if err := m.opampProxy.Stop(ctx); err != nil {
				m.log.Error(err, "failed to stop OpAMP proxy")
			}
		}
		if m.healthServer != nil {
			if err := m.healthServer.Stop(ctx); err != nil {
				m.log.Error(err, "failed to stop health listener")
			}
		}
	})
}
