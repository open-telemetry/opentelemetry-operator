// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
)

// Get the Pod DNS Policy depending on whether we're using a host network.
func GetDNSPolicy(hostNetwork bool, dnsConfig corev1.PodDNSConfig, explicitDNSPolicy *corev1.DNSPolicy) corev1.DNSPolicy {
	// If explicit DNS policy is provided, use it
	if explicitDNSPolicy != nil {
		return *explicitDNSPolicy
	}

	dnsPolicy := corev1.DNSClusterFirst
	if hostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}
	// If local DNS configuration is set, takes precedence of hostNetwork.
	if dnsConfig.Nameservers != nil {
		dnsPolicy = corev1.DNSNone
	}
	return dnsPolicy
}
