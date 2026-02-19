// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_145_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	// Use the shared migration function from v1beta1
	events := v1beta1.MigrateOTLPExporters(&otelcol.Spec.Config)

	// Record Kubernetes events for each migration
	for _, event := range events {
		existing := &corev1.ConfigMap{}
		updated := existing.DeepCopy()
		u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.145.0: %s", event.Message))
	}

	return otelcol, nil
}
