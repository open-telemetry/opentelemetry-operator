// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isMTLSEnabled(cfg config.Config, ta v1alpha1.TargetAllocator) bool {
	annotations := ta.GetAnnotations()
	if annotations == nil {
		return false
	}
	if annotations["opentelemetry.io/ta-mtls-enabled"] != "true" {
		return false
	}
	useCertManager := annotations["opentelemetry.io/ta-mtls-use-cert-manager"] != "false"
	return cfg.CertManagerAvailability == certmanager.Available && useCertManager
}
