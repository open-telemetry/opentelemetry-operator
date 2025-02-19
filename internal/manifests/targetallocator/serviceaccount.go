// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceAccountName returns the name of the existing or self-provisioned service account to use for the given instance.
func ServiceAccountName(instance v1alpha1.TargetAllocator) string {
	if len(instance.Spec.ServiceAccount) == 0 {
		return naming.TargetAllocatorServiceAccount(instance.Name)
	}

	return instance.Spec.ServiceAccount
}

// ServiceAccount returns the service account for the given instance.
func ServiceAccount(params Params) *corev1.ServiceAccount {
	if len(params.TargetAllocator.Spec.ServiceAccount) > 0 {
		return nil
	}
	name := naming.TargetAllocatorServiceAccount(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.TargetAllocator.Namespace,
			Labels:      labels,
			Annotations: params.TargetAllocator.Annotations,
		},
	}
}
