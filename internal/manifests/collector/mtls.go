// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isTAMTLSEnabledWithCertManager(cfg config.Config, ta *v1alpha1.TargetAllocator) bool {
	if ta == nil || ta.Spec.Mtls == nil || !ta.Spec.Mtls.Enabled {
		return false
	}

	useCertManager := true
	if ta.Spec.Mtls.UseCertManager != nil {
		useCertManager = *ta.Spec.Mtls.UseCertManager
	}

	return useCertManager && cfg.CertManagerAvailability == certmanager.Available
}
