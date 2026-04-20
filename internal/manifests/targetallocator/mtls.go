// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func isMTLSEnabled(cfg config.Config, collector *v1beta1.OpenTelemetryCollector) bool {
	return cfg.CertManagerAvailability == certmanager.Available &&
		collector != nil &&
		collector.Spec.TargetAllocator.Enabled &&
		collector.Spec.TargetAllocator.Mtls != nil &&
		collector.Spec.TargetAllocator.Mtls.Enabled
}
