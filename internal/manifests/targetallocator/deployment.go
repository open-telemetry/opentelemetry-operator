// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Deployment builds the deployment for the given instance.
func Deployment(params Params) (*appsv1.Deployment, error) {
	name := naming.TargetAllocator(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	configMap, err := ConfigMap(params)
	if err != nil {
		params.Log.Info("failed to construct target allocator config map for annotations")
		configMap = nil
	}
	annotations := Annotations(params.TargetAllocator, configMap, params.Config.AnnotationsFilter())

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.TargetAllocator.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: params.TargetAllocator.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.TASelectorLabels(params.TargetAllocator, ComponentOpenTelemetryTargetAllocator),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ServiceAccountName(params.TargetAllocator),
					InitContainers:                params.TargetAllocator.Spec.InitContainers,
					Containers:                    append(params.TargetAllocator.Spec.AdditionalContainers, Container(params.Config, params.Log, params.TargetAllocator)),
					Volumes:                       Volumes(params.Config, params.TargetAllocator),
					DNSPolicy:                     manifestutils.GetDNSPolicy(params.TargetAllocator.Spec.HostNetwork, params.TargetAllocator.Spec.PodDNSConfig),
					DNSConfig:                     &params.TargetAllocator.Spec.PodDNSConfig,
					HostNetwork:                   params.TargetAllocator.Spec.HostNetwork,
					ShareProcessNamespace:         &params.TargetAllocator.Spec.ShareProcessNamespace,
					Tolerations:                   params.TargetAllocator.Spec.Tolerations,
					NodeSelector:                  params.TargetAllocator.Spec.NodeSelector,
					SecurityContext:               params.TargetAllocator.Spec.PodSecurityContext,
					PriorityClassName:             params.TargetAllocator.Spec.PriorityClassName,
					Affinity:                      params.TargetAllocator.Spec.Affinity,
					TerminationGracePeriodSeconds: params.TargetAllocator.Spec.TerminationGracePeriodSeconds,
					TopologySpreadConstraints:     params.TargetAllocator.Spec.TopologySpreadConstraints,
				},
			},
		},
	}, nil
}
