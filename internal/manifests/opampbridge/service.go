// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func Service(params manifests.Params) *corev1.Service {
	name := naming.OpAMPBridgeService(params.OpAMPBridge.Name)
	labels := manifestutils.Labels(params.OpAMPBridge.ObjectMeta, name, params.OpAMPBridge.Spec.Image, ComponentOpAMPBridge, []string{})
	selector := manifestutils.SelectorLabels(params.OpAMPBridge.ObjectMeta, ComponentOpAMPBridge)

	ports := []corev1.ServicePort{{
		Name:       "opamp-bridge",
		Port:       80,
		TargetPort: intstr.FromInt(8080),
	}}

	ports = append(params.OpAMPBridge.Spec.Ports, ports...)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.OpAMPBridge.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:       selector,
			Ports:          ports,
			IPFamilies:     params.OpAMPBridge.Spec.IpFamilies,
			IPFamilyPolicy: params.OpAMPBridge.Spec.IpFamilyPolicy,
		},
	}
}
