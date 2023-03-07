package opampbridge

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func getDNSPolicy(opampBridge v1alpha1.OpAMPBridge) corev1.DNSPolicy {
	dnsPolicy := corev1.DNSClusterFirst
	if opampBridge.Spec.HostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}
	return dnsPolicy
}
