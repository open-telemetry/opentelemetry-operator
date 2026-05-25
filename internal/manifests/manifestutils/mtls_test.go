// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestIsTAMTLSEnabledDefaultsUseCertManagerToTrue(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.True(t, IsTAMTLSEnabled(cfg, ta))
}

func TestIsTAMTLSEnabledDisabledWhenUseCertManagerFalse(t *testing.T) {
	useCertManager := false
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: &useCertManager,
	}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, IsTAMTLSEnabled(cfg, ta))
}

func TestIsTAMTLSEnabledNilTA(t *testing.T) {
	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, IsTAMTLSEnabled(cfg, nil))
}

func TestIsTAMTLSEnabledNilMtls(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, IsTAMTLSEnabled(cfg, ta))
}

func TestIsTAMTLSEnabledDisabledMtls(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: false}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, IsTAMTLSEnabled(cfg, ta))
}

func TestIsTAMTLSEnabledCertManagerUnavailable(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	cfg := config.Config{CertManagerAvailability: certmanager.NotAvailable}

	assert.False(t, IsTAMTLSEnabled(cfg, ta))
}
