// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatorbridge

import (
	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opampagent "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	bridgeconfig "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/healthcheck"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

func NewManagerOpts(log logr.Logger, cfg *bridgeconfig.Config, c client.Client, _ *rest.Config) bridgemanager.ManagerOpts {
	opampClient := cfg.CreateClient()
	applier := operator.NewClient(cfg.Name, log.WithName("operator-client"), c, cfg.GetComponentsAllowed())
	opampProxy := proxy.NewOpAMPProxy(log.WithName("server"), cfg.ListenAddr)
	opampAgent := opampagent.NewAgent(log.WithName("agent"), applier, cfg, opampClient, opampProxy)
	return bridgemanager.ManagerOpts{
		Log:                     log,
		HealthServer:            healthcheck.NewServer(log.WithName("healthcheck"), cfg.HealthListenAddr),
		OpAMPProxy:              opampProxy,
		PermissionReviewClient:  c,
		ListRequiredPermissions: ListRequiredPermissions,
		Runtimes: []bridgemanager.Runtime{
			{
				Name:       cfg.Name,
				Client:     opampClient,
				OpAMPAgent: opampAgent,
			},
		},
	}
}

func ListRequiredPermissions() ([]bridgemanager.Permission, error) {
	return []bridgemanager.Permission{
		{Verb: "get", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "list", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "create", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "update", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "delete", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "get", Resource: "pods"},
		{Verb: "list", Resource: "pods"},
	}, nil
}
