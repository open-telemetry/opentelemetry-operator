// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

var ErrorDNSPolicy = errors.New("when a dnsPolicy is set to None, the dnsConfig field has to be specified")

// Get the Pod DNS Policy depending on whether we're using a host network.
func GetDNSPolicy(hostNetwork bool, dnsConfig corev1.PodDNSConfig) corev1.DNSPolicy {
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
