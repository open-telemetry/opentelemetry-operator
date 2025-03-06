// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_111_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam
	return otelcol, otelcol.Spec.Config.Service.ApplyDefaults(u.Log)
}
