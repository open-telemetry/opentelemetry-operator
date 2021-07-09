// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package naming is for determining the names for components (containers, services, ...).
package naming

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// ConfigMap builds the name for the config map used in the OpenTelemetryCollector containers.
func ConfigMap(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// LBConfigMap returns the name for the config map used in the LoadBalancer.
func LBConfigMap(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-loadbalancer", otelcol.Name)
}

// ConfigMapVolume returns the name to use for the config map's volume in the pod.
func ConfigMapVolume() string {
	return "otc-internal"
}

// LBConfigMapVolume returns the name to use for the config map's volume in the LoadBalancer pod.
func LBConfigMapVolume() string {
	return "lb-internal"
}

// Container returns the name to use for the container in the pod.
func Container() string {
	return "otc-container"
}

// LBContainer returns the name to use for the container in the LoadBalancer pod.
func LBContainer() string {
	return "lb-container"
}

// Collector builds the collector (deployment/daemonset) name based on the instance.
func Collector(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// LoadBalancer returns the LoadBalancer deployment resource name.
func LoadBalancer(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-loadbalancer", otelcol.Name)
}

// HeadlessService builds the name for the headless service based on the instance.
func HeadlessService(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-headless", Service(otelcol))
}

// MonitoringService builds the name for the monitoring service based on the instance.
func MonitoringService(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-monitoring", Service(otelcol))
}

// Service builds the service name based on the instance.
func Service(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// LBService returns the name to use for the LoadBalancer service.
func LBService(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-loadbalancer", otelcol.Name)
}

// ServiceAccount builds the service account name based on the instance.
func ServiceAccount(otelcol v1alpha1.OpenTelemetryCollector) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}
