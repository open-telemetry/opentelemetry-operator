// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isTAMTLSEnabledWithCertManager(cfg config.Config, otelcol v1beta1.OpenTelemetryCollector) bool {
	if otelcol.Spec.TargetAllocator.Mtls == nil {
		return false
	}

	useCertManager := true
	if otelcol.Spec.TargetAllocator.Mtls.UseCertManager != nil {
		useCertManager = *otelcol.Spec.TargetAllocator.Mtls.UseCertManager
	}

	return otelcol.Spec.TargetAllocator.Enabled &&
		otelcol.Spec.TargetAllocator.Mtls.Enabled &&
		useCertManager &&
		cfg.CertManagerAvailability == certmanager.Available
}
