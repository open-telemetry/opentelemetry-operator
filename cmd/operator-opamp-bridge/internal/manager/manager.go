// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"sync"
	"time"

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

type ManagerOpts struct {
	Log                     logr.Logger
	Runtimes                []Runtime
	HealthServer            *healthcheck.Server
	OpAMPProxy              *proxy.OpAMPProxy
	KubernetesClient        KubernetesClient
	PermissionReviewClient  PermissionReviewClient
	ListRequiredPermissions func() ([]Permission, error)
}

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

func New(opts ManagerOpts) *Manager {
	manager := &Manager{
		log:                     opts.Log,
		runtimes:                opts.Runtimes,
		healthServer:            opts.HealthServer,
		opampProxy:              opts.OpAMPProxy,
		kubernetesClient:        opts.KubernetesClient,
		permissionReviewClient:  opts.PermissionReviewClient,
		listRequiredPermissions: opts.ListRequiredPermissions,
	}
	return manager
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
			m.Shutdown()
			return err
		}
	}
	if m.kubernetesClient != nil {
		kubernetesClientCtx, cancelKubernetesClient := context.WithCancel(ctx)
		m.cancelKubernetesClient = cancelKubernetesClient
		if err := m.kubernetesClient.Start(kubernetesClientCtx); err != nil {
			cancelKubernetesClient()
			m.cancelKubernetesClient = nil
			m.Shutdown()
			return err
		}
	}
	if m.opampProxy != nil {
		if err := m.opampProxy.Start(); err != nil {
			m.Shutdown()
			return err
		}
	}
	if m.healthServer != nil {
		if err := m.healthServer.Start(ctx); err != nil {
			m.Shutdown()
			return err
		}
	}
	return nil
}

func (m *Manager) Shutdown() {
	m.shutdownOnce.Do(func() {
		if m.cancelKubernetesClient != nil {
			m.cancelKubernetesClient()
		}
		for _, runtime := range m.runtimes {
			runtime.OpAMPAgent.Shutdown()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
