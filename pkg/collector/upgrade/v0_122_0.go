// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_122_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam
	tel := otelcol.Spec.Config.Service.GetTelemetry()

	if tel == nil || tel.Metrics.Address == "" {
		return otelcol, nil
	}

	host, port, err := otelcol.Spec.Config.Service.MetricsEndpoint(u.Log)
	if err != nil {
		return otelcol, err
	}

	// service.telemetry.metrics.address is deprecated and should not be used anymore.
	// Setting the "address" field to an empty string explicitly removes the value
	// during Kubernetes serialization. Directly deleting the field from the map using
	// delete(metrics, "address") does not work because Kubernetes treats missing fields
	// differently from explicitly empty ones. By assigning "", we ensure the configuration
	// is updated correctly when the resource is persisted.
	tel.Metrics.Address = ""
	reader := v1beta1.AddPrometheusMetricsEndpoint(host, port)
	tel.Metrics.Readers = append(tel.Metrics.Readers, reader)

	otelcol.Spec.Config.Service.Telemetry, err = tel.ToAnyConfig()
	if err != nil {
		return otelcol, err
	}

	return otelcol, nil
}
