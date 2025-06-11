// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/opampbridge"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestConfigChangesOnAutoDetect(t *testing.T) {
	// prepare
	mock := &mockAutoDetect{
		OpenShiftRoutesAvailabilityFunc: func() (openshift.RoutesAvailability, error) {
			return openshift.RoutesAvailable, nil
		},
		PrometheusCRsAvailabilityFunc: func() (prometheus.Availability, error) {
			return prometheus.Available, nil
		},
		RBACPermissionsFunc: func(ctx context.Context) (rbac.Availability, error) {
			return rbac.Available, nil
		},
		CertManagerAvailabilityFunc: func(ctx context.Context) (certmanager.Availability, error) {
			return certmanager.Available, nil
		},
		TargetAllocatorAvailabilityFunc: func() (targetallocator.Availability, error) {
			return targetallocator.Available, nil
		},
		CollectorAvailabilityFunc: func() (collector.Availability, error) {
			return collector.Available, nil
		},
		OpAmpBridgeAvailabilityFunc: func() (opampbridge.Availability, error) {
			return opampbridge.Available, nil
		},
	}
	cfg := config.New()

	// sanity check
	require.Equal(t, openshift.RoutesNotAvailable, cfg.OpenShiftRoutesAvailability)
	require.Equal(t, prometheus.NotAvailable, cfg.PrometheusCRAvailability)
	require.Equal(t, rbac.NotAvailable, cfg.CreateRBACPermissions)
	require.Equal(t, certmanager.NotAvailable, cfg.CertManagerAvailability)
	require.Equal(t, targetallocator.NotAvailable, cfg.TargetAllocatorAvailability)
	require.Equal(t, collector.NotAvailable, cfg.CollectorAvailability)
	require.Equal(t, opampbridge.NotAvailable, cfg.OpAmpBridgeAvailability)

	// test
	require.NoError(t, autodetect.ApplyAutoDetect(mock, &cfg, logr.Discard()))

	// verify
	assert.Equal(t, openshift.RoutesAvailable, cfg.OpenShiftRoutesAvailability)
	require.Equal(t, prometheus.Available, cfg.PrometheusCRAvailability)
	require.Equal(t, rbac.Available, cfg.CreateRBACPermissions)
	require.Equal(t, certmanager.Available, cfg.CertManagerAvailability)
	require.Equal(t, targetallocator.Available, cfg.TargetAllocatorAvailability)
	require.Equal(t, opampbridge.Available, cfg.OpAmpBridgeAvailability)
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	OpenShiftRoutesAvailabilityFunc func() (openshift.RoutesAvailability, error)
	PrometheusCRsAvailabilityFunc   func() (prometheus.Availability, error)
	RBACPermissionsFunc             func(ctx context.Context) (rbac.Availability, error)
	CertManagerAvailabilityFunc     func(ctx context.Context) (certmanager.Availability, error)
	TargetAllocatorAvailabilityFunc func() (targetallocator.Availability, error)
	CollectorAvailabilityFunc       func() (collector.Availability, error)
	OpAmpBridgeAvailabilityFunc     func() (opampbridge.Availability, error)
}

func (m *mockAutoDetect) OpAmpBridgeAvailablity() (opampbridge.Availability, error) {
	if m.OpAmpBridgeAvailabilityFunc != nil {
		return m.OpAmpBridgeAvailabilityFunc()
	}
	return opampbridge.NotAvailable, nil
}

func (m *mockAutoDetect) FIPSEnabled(_ context.Context) bool {
	return false
}

func (m *mockAutoDetect) OpenShiftRoutesAvailability() (openshift.RoutesAvailability, error) {
	if m.OpenShiftRoutesAvailabilityFunc != nil {
		return m.OpenShiftRoutesAvailabilityFunc()
	}
	return openshift.RoutesNotAvailable, nil
}

func (m *mockAutoDetect) PrometheusCRsAvailability() (prometheus.Availability, error) {
	if m.PrometheusCRsAvailabilityFunc != nil {
		return m.PrometheusCRsAvailabilityFunc()
	}
	return prometheus.NotAvailable, nil
}

func (m *mockAutoDetect) RBACPermissions(ctx context.Context) (rbac.Availability, error) {
	if m.RBACPermissionsFunc != nil {
		return m.RBACPermissionsFunc(ctx)
	}
	return rbac.NotAvailable, nil
}

func (m *mockAutoDetect) CertManagerAvailability(ctx context.Context) (certmanager.Availability, error) {
	if m.CertManagerAvailabilityFunc != nil {
		return m.CertManagerAvailabilityFunc(ctx)
	}
	return certmanager.NotAvailable, nil
}

func (m *mockAutoDetect) TargetAllocatorAvailability() (targetallocator.Availability, error) {
	if m.TargetAllocatorAvailabilityFunc != nil {
		return m.TargetAllocatorAvailabilityFunc()
	}
	return targetallocator.NotAvailable, nil
}

func (m *mockAutoDetect) CollectorAvailability() (collector.Availability, error) {
	if m.CollectorAvailabilityFunc != nil {
		return m.CollectorAvailabilityFunc()
	}
	return collector.NotAvailable, nil
}
