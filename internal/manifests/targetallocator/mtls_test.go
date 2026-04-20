// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestIsMTLSEnabledDefaultsUseCertManagerToTrue(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Annotations = map[string]string{
		"opentelemetry.io/ta-mtls-enabled": "true",
	}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.True(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledDisabledWhenUseCertManagerFalse(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Annotations = map[string]string{
		"opentelemetry.io/ta-mtls-enabled":          "true",
		"opentelemetry.io/ta-mtls-use-cert-manager": "false",
	}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledNilMtlsNoAnnotation(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Annotations = nil

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledDisabledMtlsNoAnnotation(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Annotations = nil

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isMTLSEnabled(cfg, ta))
}

func TestIsMTLSEnabledNoAnnotationCertManagerUnavailable(t *testing.T) {
	ta := v1alpha1.TargetAllocator{}
	ta.Annotations = nil

	cfg := config.Config{CertManagerAvailability: certmanager.NotAvailable}

	assert.False(t, isMTLSEnabled(cfg, ta))
}
