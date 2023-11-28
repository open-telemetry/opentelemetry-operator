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

// ConfigMap builds the name for the config map used in the OpenTelemetryCollector containers.
func ConfigMap(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// TAConfigMap returns the name for the config map used in the TargetAllocator.
func TAConfigMap(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// OpAMPBridgeConfigMap builds the name for the config map used in the OpAMPBridge containers.
func OpAMPBridgeConfigMap(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}

// ConfigMapVolume returns the name to use for the config map's volume in the pod.
func ConfigMapVolume() string {
	return "otc-internal"
}

// ConfigMapExtra returns the prefix to use for the extras mounted configmaps in the pod.
func ConfigMapExtra(extraConfigMapName string) string {
	return DNSName(Truncate("configmap-%s", 63, extraConfigMapName))
}

// TAConfigMapVolume returns the name to use for the config map's volume in the TargetAllocator pod.
func TAConfigMapVolume() string {
	return "ta-internal"
}

// OpAMPBridgeConfigMapVolume returns the name to use for the config map's volume in the OpAMPBridge pod.
func OpAMPBridgeConfigMapVolume() string {
	return "opamp-bridge-internal"
}

// Container returns the name to use for the container in the pod.
func Container() string {
	return "otc-container"
}

// TAContainer returns the name to use for the container in the TargetAllocator pod.
func TAContainer() string {
	return "ta-container"
}

// OpAMPBridgeContainer returns the name to use for the container in the OpAMPBridge pod.
func OpAMPBridgeContainer() string {
	return "opamp-bridge-container"
}

// Collector builds the collector (deployment/daemonset) name based on the instance.
func Collector(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// HorizontalPodAutoscaler builds the autoscaler name based on the instance.
func HorizontalPodAutoscaler(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// HorizontalPodAutoscaler builds the autoscaler name based on the instance.
func PodDisruptionBudget(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// OpenTelemetryCollector builds the collector (deployment/daemonset) name based on the instance.
func OpenTelemetryCollector(otelcol string) string {
	return DNSName(Truncate("%s", 63, otelcol))
}

// OpenTelemetryCollectorName builds the collector (deployment/daemonset) name based on the instance.
func OpenTelemetryCollectorName(otelcolName string) string {
	return DNSName(Truncate("%s", 63, otelcolName))
}

// TargetAllocator returns the TargetAllocator deployment resource name.
func TargetAllocator(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// OpAMPBridge returns the OpAMPBridge deployment resource name.
func OpAMPBridge(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}

// HeadlessService builds the name for the headless service based on the instance.
func HeadlessService(otelcol string) string {
	return DNSName(Truncate("%s-headless", 63, Service(otelcol)))
}

// MonitoringService builds the name for the monitoring service based on the instance.
func MonitoringService(otelcol string) string {
	return DNSName(Truncate("%s-monitoring", 63, Service(otelcol)))
}

// Service builds the service name based on the instance.
func Service(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// Ingress builds the ingress name based on the instance.
func Ingress(otelcol string) string {
	return DNSName(Truncate("%s-ingress", 63, otelcol))
}

// Route builds the route name based on the instance.
func Route(otelcol string, prefix string) string {
	return DNSName(Truncate("%s-%s-route", 63, prefix, otelcol))
}
// ClusterRole builds the cluster role name based on the instance.
func ClusterRole(otelcol string, namespace string) string {
	return DNSName(Truncate("%s-%s-cluster-role", 63, otelcol, namespace))
}

// ClusterRoleBinding builds the cluster role binding name based on the instance.
func ClusterRoleBinding(otelcol string) string {
	return DNSName(Truncate("%s-cluster-role-binding", 63, otelcol))
}

// TAService returns the name to use for the TargetAllocator service.
func TAService(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// OpAMPBridgeService returns the name to use for the OpAMPBridge service.
func OpAMPBridgeService(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}

// ServiceAccount builds the service account name based on the instance.
func ServiceAccount(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// ServiceMonitor builds the service account name based on the instance.
func ServiceMonitor(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// TargetAllocatorServiceAccount returns the TargetAllocator service account resource name.
func TargetAllocatorServiceAccount(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// OpAMPBridgeServiceAccount builds the service account name based on the instance.
func OpAMPBridgeServiceAccount(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}
