// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/otelconfig"
)

func upgrade0_111_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	return otelcol, applyDefaults(otelcol, u.Log)
}

func applyDefaults(otelcol *v1beta1.OpenTelemetryCollector, logger logr.Logger) error {
	if tel := otelconfig.GetTelemetry(&otelcol.Spec.Config.Service, logger); tel != nil && len(tel.Metrics.Readers) > 0 {
		// service.telemetry.metrics.readers is already configured; setting the deprecated
		// address field here would duplicate that endpoint. A later upgrade step migrates
		// address to readers and would then add a second Prometheus reader for it, making
		// the collector fail at startup with "address already in use".
		return nil
	}

	telemetryAddr, telemetryPort, err := otelconfig.MetricsEndpoint(&otelcol.Spec.Config.Service, logger)
	if err != nil {
		return err
	}

	tm := &v1beta1.AnyConfig{
		Object: map[string]any{
			"metrics": map[string]any{
				"address": fmt.Sprintf("%s:%d", telemetryAddr, telemetryPort),
			},
		},
	}

	if otelcol.Spec.Config.Service.Telemetry == nil {
		otelcol.Spec.Config.Service.Telemetry = tm
		return nil
	}
	// NOTE: Merge without overwrite. If a telemetry endpoint is specified, the defaulting
	// respects the configuration and returns an equal value.
	if err := mergo.Merge(otelcol.Spec.Config.Service.Telemetry, tm); err != nil {
		return fmt.Errorf("telemetry config merge failed: %w", err)
	}
	return nil
}
