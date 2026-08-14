// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	otelConfig "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/otelconfig"
)

func upgrade0_122_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	tel := otelconfig.GetTelemetry(&otelcol.Spec.Config.Service, u.Log)

	if tel == nil || tel.Metrics.Address == "" {
		return otelcol, nil
	}

	host, port, err := otelconfig.MetricsEndpoint(&otelcol.Spec.Config.Service, u.Log)
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
	if !hasPrometheusReader(tel.Metrics.Readers, host, port) {
		reader := otelconfig.AddPrometheusMetricsEndpoint(host, port)
		tel.Metrics.Readers = append(tel.Metrics.Readers, reader)
	}

	otelcol.Spec.Config.Service.Telemetry, err = otelconfig.TelemetryToAnyConfig(tel)
	if err != nil {
		return otelcol, err
	}

	return otelcol, nil
}

// hasPrometheusReader reports whether readers already contains a Prometheus pull
// reader bound to the given host:port, so migrating the deprecated address field
// doesn't add a second reader for the same endpoint.
func hasPrometheusReader(readers []otelConfig.MetricReader, host string, port int32) bool {
	for _, r := range readers {
		if r.Pull == nil || r.Pull.Exporter.Prometheus == nil {
			continue
		}
		prom := r.Pull.Exporter.Prometheus
		if prom.Host != nil && *prom.Host == host && prom.Port != nil && *prom.Port == int(port) {
			return true
		}
	}
	return false
}
