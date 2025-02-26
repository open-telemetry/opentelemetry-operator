// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestNewConfig(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithCollectorImage("some-image"),
		config.WithCollectorConfigMapEntry("some-config.yaml"),
		config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
		config.WithPrometheusCRAvailability(prometheus.Available),
	)

	// test
	assert.Equal(t, "some-image", cfg.CollectorImage())
	assert.Equal(t, "some-config.yaml", cfg.CollectorConfigMapEntry())
	assert.Equal(t, openshift.RoutesAvailable, cfg.OpenShiftRoutesAvailability())
	assert.Equal(t, prometheus.Available, cfg.PrometheusCRAvailability())
}

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
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
	)

	// sanity check
	require.Equal(t, openshift.RoutesNotAvailable, cfg.OpenShiftRoutesAvailability())
	require.Equal(t, prometheus.NotAvailable, cfg.PrometheusCRAvailability())
	require.Equal(t, rbac.NotAvailable, cfg.CreateRBACPermissions())
	require.Equal(t, certmanager.NotAvailable, cfg.CertManagerAvailability())
	require.Equal(t, targetallocator.NotAvailable, cfg.TargetAllocatorAvailability())

	// test
	err := cfg.AutoDetect()
	require.NoError(t, err)

	// verify
	assert.Equal(t, openshift.RoutesAvailable, cfg.OpenShiftRoutesAvailability())
	require.Equal(t, prometheus.Available, cfg.PrometheusCRAvailability())
	require.Equal(t, rbac.Available, cfg.CreateRBACPermissions())
	require.Equal(t, certmanager.Available, cfg.CertManagerAvailability())
	require.Equal(t, targetallocator.Available, cfg.TargetAllocatorAvailability())
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	OpenShiftRoutesAvailabilityFunc func() (openshift.RoutesAvailability, error)
	PrometheusCRsAvailabilityFunc   func() (prometheus.Availability, error)
	RBACPermissionsFunc             func(ctx context.Context) (rbac.Availability, error)
	CertManagerAvailabilityFunc     func(ctx context.Context) (certmanager.Availability, error)
	TargetAllocatorAvailabilityFunc func() (targetallocator.Availability, error)
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
