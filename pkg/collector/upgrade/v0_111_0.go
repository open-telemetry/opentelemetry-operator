// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_111_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam

	return otelcol, applyDefaults(otelcol, u.Log)
}

func applyDefaults(otelcol *v1beta1.OpenTelemetryCollector, logger logr.Logger) error {
	telemetryAddr, telemetryPort, err := otelcol.Spec.Config.Service.MetricsEndpoint(logger)
	if err != nil {
		return err
	}

	tm := &v1beta1.AnyConfig{
		Object: map[string]interface{}{
			"metrics": map[string]interface{}{
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
