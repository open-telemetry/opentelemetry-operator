// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

type certManagerTestManager struct {
	ctrl.Manager
	runnable manager.Runnable
}

func (m *certManagerTestManager) Add(r manager.Runnable) error {
	m.runnable = r
	return nil
}

type certManagerTestDetector struct {
	autodetect.AutoDetect
}

func (certManagerTestDetector) CertManagerAvailability(context.Context) (certmanager.Availability, error) {
	return certmanager.Available, nil
}

func TestSetupCertManagerAvailabilityWatcher(t *testing.T) {
	for _, standalone := range []bool{false, true} {
		name := "controller"
		if standalone {
			name = "standalone webhook"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			mgr := &certManagerTestManager{}
			result := &SetupResult{Manager: mgr, Autodetector: certManagerTestDetector{}, Config: config.New()}
			stop := cancel
			if standalone {
				stop = nil
			}
			require.NoError(t, SetupCertManagerAvailabilityWatcher(result, stop))
			require.NotNil(t, mgr.runnable)
			cfg := result.Config
			done := make(chan error, 1)
			go func() { done <- mgr.runnable.Start(ctx) }()
			if standalone {
				require.Eventually(t, func() bool {
					return cfg.GetCertManagerAvailability() == certmanager.Available
				}, 5*time.Second, time.Millisecond)
				assert.NoError(t, ctx.Err())
				cancel()
			}
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("watcher did not stop")
			}
			assert.Equal(t, certmanager.NotAvailable, cfg.CertManagerAvailability)
			if !standalone {
				assert.Nil(t, cfg.Internal.CertManagerAvailability)
				assert.ErrorIs(t, ctx.Err(), context.Canceled)
			}
		})
	}
}

func TestSetupCertManagerAvailabilityWatcherDisabled(t *testing.T) {
	gate := featuregate.EnableCertManagerAvailabilityWatcher
	previous := gate.IsEnabled()
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(gate.ID(), false))
	t.Cleanup(func() { require.NoError(t, colfeaturegate.GlobalRegistry().Set(gate.ID(), previous)) })
	mgr := &certManagerTestManager{}
	result := &SetupResult{Manager: mgr, Config: config.New()}
	require.NoError(t, SetupCertManagerAvailabilityWatcher(result, nil))
	assert.Nil(t, mgr.runnable)
	assert.Nil(t, result.Config.Internal.CertManagerAvailability)
}
