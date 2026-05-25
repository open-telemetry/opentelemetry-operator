// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestIsMTLSEnabledDefaultsUseCertManagerToTrue(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.True(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledDisabledWhenUseCertManagerFalse(t *testing.T) {
	useCertManager := false
	ta := v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: &useCertManager,
	}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledNilMtls(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledDisabledMtls(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: false}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledCertManagerUnavailable(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	cfg := config.Config{CertManagerAvailability: certmanager.NotAvailable}

	assert.False(t, isMTLSEnabled(cfg, ta))
}
