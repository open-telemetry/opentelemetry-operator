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

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
)

// ConfigMap builds the name for the config map used in the SplunkOtelAgent containers.
func ConfigMap(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// TAConfigMap returns the name for the config map used in the TargetAllocator.
func TAConfigMap(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-targetallocator", otelcol.Name)
}

// ConfigMapVolume returns the name to use for the config map's volume in the pod.
func ConfigMapVolume() string {
	return "otc-internal"
}

// TAConfigMapVolume returns the name to use for the config map's volume in the TargetAllocator pod.
func TAConfigMapVolume() string {
	return "ta-internal"
}

// Container returns the name to use for the container in the pod.
func Container() string {
	return "otc-container"
}

// TAContainer returns the name to use for the container in the TargetAllocator pod.
func TAContainer() string {
	return "ta-container"
}

// Collector builds the collector (deployment/daemonset) name based on the instance.
func Collector(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// TargetAllocator returns the TargetAllocator deployment resource name.
func TargetAllocator(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-targetallocator", otelcol.Name)
}

// HeadlessService builds the name for the headless service based on the instance.
func HeadlessService(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-headless", Service(otelcol))
}

// MonitoringService builds the name for the monitoring service based on the instance.
func MonitoringService(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-monitoring", Service(otelcol))
}

// Service builds the service name based on the instance.
func Service(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// TAService returns the name to use for the TargetAllocator service.
func TAService(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-targetallocator", otelcol.Name)
}

// ServiceAccount builds the service account name based on the instance.
func ServiceAccount(otelcol v1alpha1.SplunkOtelAgent) string {
	return fmt.Sprintf("%s-collector", otelcol.Name)
}

// Namespace builds the namespace name based on the instance.
func Namespace(otelcol v1alpha1.SplunkOtelAgent) string {
	return "splunk-otel-operator-system"
}
