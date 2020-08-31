package naming

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// ConfigMap builds the name for the config map used in the OpenTelemetryCollector containers
func ConfigMap(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// ConfigMapVolume returns the name to use for the config map's volume in the pod
func ConfigMapVolume() string {
	return "otc-internal"
}

// Container returns the name to use for the container in the pod
func Container() string {
	return "otc-container"
}

// Collector builds the collector (deployment/daemonset) name based on the instance
func Collector(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// HeadlessService builds the name for the headless service based on the instance
func HeadlessService(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-headless", Service(otelcol))
}

// MonitoringService builds the name for the monitoring service based on the instance
func MonitoringService(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-monitoring", Service(otelcol))
}

// Service builds the service name based on the instance
func Service(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// ServiceAccount builds the service account name based on the instance
func ServiceAccount(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}
