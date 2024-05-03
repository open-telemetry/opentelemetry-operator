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

package collector

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var ErrorDNSPolicy = errors.New("when a dnsPolicy is set to None, the dnsConfig field has to be specified")

func getDNSPolicy(otelcol v1beta1.OpenTelemetryCollector) corev1.DNSPolicy {
	dnsPolicy := otelcol.Spec.PodDNSPolicy
	if otelcol.Spec.HostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}
	if otelcol.Spec.PodDNSConfig.Nameservers != nil {
		dnsPolicy = corev1.DNSNone
	}
	return dnsPolicy
}
