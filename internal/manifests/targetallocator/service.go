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

package targetallocator

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func Service(params Params) *corev1.Service {
	name := naming.TAService(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)
	selector := manifestutils.TASelectorLabels(params.TargetAllocator, ComponentOpenTelemetryTargetAllocator)

	ports := make([]corev1.ServicePort, 0)
	ports = append(ports, corev1.ServicePort{
		Name:       "targetallocation",
		Port:       80,
		TargetPort: intstr.FromString("http")})

	if params.Config.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		ports = append(ports, corev1.ServicePort{
			Name:       "targetallocation-https",
			Port:       443,
			TargetPort: intstr.FromString("https")})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TAService(params.TargetAllocator.Name),
			Namespace: params.TargetAllocator.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:       selector,
			Ports:          ports,
			IPFamilies:     params.TargetAllocator.Spec.IpFamilies,
			IPFamilyPolicy: params.TargetAllocator.Spec.IpFamilyPolicy,
		},
	}
}
