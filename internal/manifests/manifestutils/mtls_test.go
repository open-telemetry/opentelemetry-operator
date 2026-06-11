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

func TestIsTAMTLSEnabledTrue(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	assert.True(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSEnabledFalse(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}
	ta.Spec.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: false}

	assert.False(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSEnabledNilTA(t *testing.T) {
	assert.False(t, IsTAMTLSEnabled(nil))
}

func TestIsTAMTLSEnabledNilMtls(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{}

	assert.False(t, IsTAMTLSEnabled(ta))
}

func TestIsTAMTLSCertManagerEnabled(t *testing.T) {
	boolTrue := true
	boolFalse := false

	tests := []struct {
		name     string
		ta       *v1alpha1.TargetAllocator
		cfg      config.Config
		expected bool
	}{
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager defaulting to true",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: true,
		},
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager explicitly true",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: &boolTrue}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: true,
		},
		{
			name:     "mTLS enabled, cert-manager available, UseCertManager false",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true, UseCertManager: &boolFalse}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
		{
			name:     "mTLS enabled, cert-manager not available",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.NotAvailable},
			expected: false,
		},
		{
			name:     "mTLS disabled",
			ta:       &v1alpha1.TargetAllocator{Spec: v1alpha1.TargetAllocatorSpec{Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: false}}},
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
		{
			name:     "nil TA",
			ta:       nil,
			cfg:      config.Config{CertManagerAvailability: certmanager.Available},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTAMTLSCertManagerEnabled(tt.ta, tt.cfg))
		})
	}
}
