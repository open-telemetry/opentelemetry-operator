// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_130_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam

	tel := otelcol.Spec.Config.Service.GetTelemetry()

	if tel == nil || len(tel.Metrics.Readers) == 0 {
		return otelcol, nil
	}

	for i, reader := range tel.Metrics.Readers {
		if reader.Pull != nil && reader.Pull.Exporter.Prometheus.WithoutUnits == nil {
			t := true
			tel.Metrics.Readers[i].Pull.Exporter.Prometheus.WithoutUnits = &t
		}
	}

	var err error
	otelcol.Spec.Config.Service.Telemetry, err = tel.ToAnyConfig()
	if err != nil {
		return otelcol, err
	}

	return otelcol, nil
}
