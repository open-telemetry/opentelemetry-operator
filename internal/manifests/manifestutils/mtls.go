// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func IsTAMTLSEnabled(ta *v1alpha1.TargetAllocator) bool {
	return ta != nil && ta.Spec.Mtls != nil && ta.Spec.Mtls.Enabled
}

func IsTAMTLSCertManagerEnabled(ta *v1alpha1.TargetAllocator, cfg config.Config) bool {
	if !IsTAMTLSEnabled(ta) {
		return false
	}
	if ta.Spec.Mtls.UseCertManager != nil && !*ta.Spec.Mtls.UseCertManager {
		return false
	}
	return cfg.CertManagerAvailability == certmanager.Available
}
