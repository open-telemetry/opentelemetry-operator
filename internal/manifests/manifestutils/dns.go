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
