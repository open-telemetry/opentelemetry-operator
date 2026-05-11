// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestIsTAMTLSEnabledWithCertManagerDefaultsToTrue(t *testing.T) {
	collector := v1beta1.OpenTelemetryCollector{}
	collector.Spec.TargetAllocator.Enabled = true
	collector.Spec.TargetAllocator.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.True(t, isTAMTLSEnabledWithCertManager(cfg, collector))
}

func TestIsTAMTLSEnabledWithCertManagerDisabledWhenUseCertManagerFalse(t *testing.T) {
	useCertManager := false
	collector := v1beta1.OpenTelemetryCollector{}
	collector.Spec.TargetAllocator.Enabled = true
	collector.Spec.TargetAllocator.Mtls = &v1beta1.TargetAllocatorMTLS{
		Enabled:        true,
		UseCertManager: &useCertManager,
	}

	cfg := config.Config{CertManagerAvailability: certmanager.Available}

	assert.False(t, isTAMTLSEnabledWithCertManager(cfg, collector))
}
