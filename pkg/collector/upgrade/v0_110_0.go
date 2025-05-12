// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_110_0(_ VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam
	envVarExpansionFeatureFlag := "-component.UseLocalHostAsDefaultHost"
	otelcol.Spec.OpenTelemetryCommonFields.Args = RemoveFeatureGate(otelcol.Spec.OpenTelemetryCommonFields.Args, envVarExpansionFeatureFlag)
	return otelcol, nil
}
