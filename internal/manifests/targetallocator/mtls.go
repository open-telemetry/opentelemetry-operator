// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isMTLSEnabled(cfg config.Config, targetAllocator v1alpha1.TargetAllocator) bool {
	return cfg.CertManagerAvailability == certmanager.Available &&
		targetAllocator.Spec.Mtls != nil &&
		targetAllocator.Spec.Mtls.Enabled
}
