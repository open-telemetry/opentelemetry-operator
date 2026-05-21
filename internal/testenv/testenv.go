// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package testenv provides utilities for envtest-based integration tests,
// reducing boilerplate in TestMain functions across the operator test suites.
package testenv

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlenvtest "sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Environment holds the running envtest.Environment together with the REST
// configuration and a ready-to-use Kubernetes client.
type Environment struct {
	// Env is the underlying controller-runtime envtest.Environment.
	// Use it to access WebhookInstallOptions after starting.
	Env *ctrlenvtest.Environment
	// Config is the REST configuration for the running test cluster.
	Config *rest.Config
	// Client is a Kubernetes client connected to the test cluster.
	Client client.Client
}

// Start configures env with standard defaults, starts it, and returns an
// Environment with a Kubernetes client created using scheme.
//
// The following defaults are applied to env before starting:
//   - DownloadBinaryAssets is set to true.
//   - BinaryAssetsDirectory is populated from SetupEnvtestDefaultBinaryAssetsDirectory.
//   - The kube-apiserver advertise-address is set to 127.0.0.1 so that sandbox
//     environments without a default network route work correctly.
//
// On any failure Start prints a message and calls os.Exit(1), matching
// TestMain conventions.
func Start(env *ctrlenvtest.Environment, scheme *runtime.Scheme) *Environment {
	binaryAssetsDir, err := ctrlenvtest.SetupEnvtestDefaultBinaryAssetsDirectory()
	if err != nil {
		fmt.Printf("failed to find setup-envtest assets directory, using a temporary one: %v\n", err)
	}
	env.DownloadBinaryAssets = true
	env.BinaryAssetsDirectory = binaryAssetsDir
	// In sandbox environments the network namespace has no default route, so
	// kube-apiserver cannot auto-detect its advertise address. Set it explicitly.
	env.ControlPlane.GetAPIServer().Configure().Set("advertise-address", "127.0.0.1")

	cfg, err := env.Start()
	if err != nil {
		fmt.Printf("failed to start testEnv: %v\n", err)
		os.Exit(1)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("failed to setup a Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	return &Environment{
		Env:    env,
		Config: cfg,
		Client: k8sClient,
	}
}

// Stop stops the test environment. On failure it prints a message and calls
// os.Exit(1), matching TestMain conventions.
func (e *Environment) Stop() {
	if err := e.Env.Stop(); err != nil {
		fmt.Printf("failed to stop testEnv: %v\n", err)
		os.Exit(1)
	}
}

// NewWebhookManager creates a controller-runtime Manager configured to serve
// the webhook server described by opts. Metrics are disabled (BindAddress "0")
// and leader election is turned off, which is appropriate for test environments.
//
// On failure it prints a message and calls os.Exit(1).
func NewWebhookManager(cfg *rest.Config, scheme *runtime.Scheme, opts *ctrlenvtest.WebhookInstallOptions) ctrl.Manager {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    opts.LocalServingHost,
			Port:    opts.LocalServingPort,
			CertDir: opts.LocalServingCertDir,
		}),
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		fmt.Printf("failed to create webhook manager: %v\n", err)
		os.Exit(1)
	}
	return mgr
}

// RunWebhookServer starts mgr in a background goroutine and blocks until the
// embedded webhook server is ready to accept TLS connections.
// On any failure it prints a message and calls os.Exit(1).
func RunWebhookServer(ctx context.Context, mgr ctrl.Manager, opts *ctrlenvtest.WebhookInstallOptions) {
	go func() {
		if err := mgr.Start(ctx); err != nil {
			fmt.Printf("failed to start manager: %v\n", err)
			os.Exit(1)
		}
	}()

	addrPort := fmt.Sprintf("%s:%d", opts.LocalServingHost, opts.LocalServingPort)
	dialer := &net.Dialer{Timeout: time.Second}

	err := retry.OnError(wait.Backoff{
		Steps:    20,
		Duration: 10 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.1,
		Cap:      30 * time.Second,
	}, func(error) bool {
		return true
	}, func() error {
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
		conn, tlsErr := tlsDialer.DialContext(ctx, "tcp", addrPort)
		if tlsErr != nil {
			return tlsErr
		}
		_ = conn.Close()
		return nil
	})
	if err != nil {
		fmt.Printf("failed to wait for webhook server to be ready: %v\n", err)
		os.Exit(1)
	}
}
