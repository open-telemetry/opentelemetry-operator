// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package naming is for determining the names for components (containers, services, ...).
package naming

// ConfigMap builds the name for the config map used in the OpenTelemetryCollector containers.
// The configHash should be calculated using manifestutils.GetConfigMapSHA.
func ConfigMap(otelcol, configHash string) string {
	return DNSName(Truncate("%s-collector-%s", 63, otelcol, configHash[:8]))
}

// TAConfigMap returns the name for the config map used in the TargetAllocator.
func TAConfigMap(targetAllocator string) string {
	return DNSName(Truncate("%s-targetallocator", 63, targetAllocator))
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

// PodDisruptionBudget builds the pdb name based on the instance.
func PodDisruptionBudget(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// TAPodDisruptionBudget builds the pdb name based on the instance.
func TAPodDisruptionBudget(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
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

// ExtensionService builds the name for the extension service based on the instance.
func ExtensionService(otelcol string) string {
	return DNSName(Truncate("%s-extension", 63, Service(otelcol)))
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
func ClusterRoleBinding(otelcol, namespace string) string {
	return DNSName(Truncate("%s-%s-collector", 63, otelcol, namespace))
}

// Role builds the role name based on the instance.
func Role(otelcol string, roleName string) string {
	return DNSName(Truncate("%s-%s-role", 63, otelcol, roleName))
}

// RoleBinding builds the role binding name based on the instance.
func RoleBinding(otelcol, roleName string) string {
	return DNSName(Truncate("%s-%s-role-binding", 63, otelcol, roleName))
}

// TAService returns the name to use for the TargetAllocator service.
func TAService(taName string) string {
	return DNSName(Truncate("%s-targetallocator", 63, taName))
}

// OpAMPBridgeService returns the name to use for the OpAMPBridge service.
func OpAMPBridgeService(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}

// ServiceAccount builds the service account name based on the instance.
func ServiceAccount(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// ServiceMonitor builds the service Monitor name based on the instance.
func ServiceMonitor(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// PodMonitor builds the pod Monitor name based on the instance.
func PodMonitor(otelcol string) string {
	return DNSName(Truncate("%s-collector", 63, otelcol))
}

// TargetAllocatorServiceAccount returns the TargetAllocator service account resource name.
func TargetAllocatorServiceAccount(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// TargetAllocatorServiceMonitor returns the TargetAllocator service account resource name.
func TargetAllocatorServiceMonitor(otelcol string) string {
	return DNSName(Truncate("%s-targetallocator", 63, otelcol))
}

// OpAMPBridgeServiceAccount builds the service account name based on the instance.
func OpAMPBridgeServiceAccount(opampBridge string) string {
	return DNSName(Truncate("%s-opamp-bridge", 63, opampBridge))
}

// SelfSignedIssuer returns the SelfSigned Issuer name based on the instance.
func SelfSignedIssuer(otelcol string) string {
	return DNSName(Truncate("%s-self-signed-issuer", 63, otelcol))
}

// CAIssuer returns the CA Issuer name based on the instance.
func CAIssuer(otelcol string) string {
	return DNSName(Truncate("%s-ca-issuer", 63, otelcol))
}

// CACertificateSecret returns the Secret name based on the instance.
func CACertificate(otelcol string) string {
	return DNSName(Truncate("%s-ca-cert", 63, otelcol))
}

// TAServerCertificate returns the Certificate name based on the instance.
func TAServerCertificate(otelcol string) string {
	return DNSName(Truncate("%s-ta-server-cert", 63, otelcol))
}

// TAServerCertificateSecretName returns the Secret name based on the instance.
func TAServerCertificateSecretName(otelcol string) string {
	return DNSName(Truncate("%s-ta-server-cert", 63, otelcol))
}

// TAClientCertificate returns the Certificate name based on the instance.
func TAClientCertificate(otelcol string) string {
	return DNSName(Truncate("%s-ta-client-cert", 63, otelcol))
}

// TAClientCertificateSecretName returns the Secret name based on the instance.
func TAClientCertificateSecretName(otelcol string) string {
	return DNSName(Truncate("%s-ta-client-cert", 63, otelcol))
}
