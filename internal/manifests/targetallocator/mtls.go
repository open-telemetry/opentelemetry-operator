// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isMTLSEnabled(cfg config.Config, ta v1alpha1.TargetAllocator) bool {
	if ta.Spec.Mtls == nil || !ta.Spec.Mtls.Enabled {
		return false
	}
	useCertManager := true
	if ta.Spec.Mtls.UseCertManager != nil {
		useCertManager = *ta.Spec.Mtls.UseCertManager
	}
	return cfg.CertManagerAvailability == certmanager.Available && useCertManager
}
