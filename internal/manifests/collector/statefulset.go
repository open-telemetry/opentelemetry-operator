// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// StatefulSet builds the statefulset for the given instance.
func StatefulSet(params manifests.Params) (*appsv1.StatefulSet, error) {
	name := naming.Collector(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
	if err != nil {
		return nil, err
	}

	podAnnotations, err := manifestutils.PodAnnotations(params.OtelCol, params.Config.AnnotationsFilter)
	if err != nil {
		return nil, err
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: statefulSetServiceName(params.OtelCol),
			Selector: &metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ServiceAccountName(params.OtelCol),
					InitContainers:                params.OtelCol.Spec.InitContainers,
					Containers:                    append([]corev1.Container{Container(params.Config, params.Log, params.OtelCol, true, params.TargetAllocator)}, params.OtelCol.Spec.AdditionalContainers...),
					Volumes:                       Volumes(params.Config, params.OtelCol, params.TargetAllocator),
					DNSPolicy:                     manifestutils.GetDNSPolicy(params.OtelCol.Spec.HostNetwork, params.OtelCol.Spec.PodDNSConfig, params.OtelCol.Spec.DNSPolicy),
					DNSConfig:                     &params.OtelCol.Spec.PodDNSConfig,
					HostNetwork:                   params.OtelCol.Spec.HostNetwork,
					HostPID:                       params.OtelCol.Spec.HostPID,
					HostUsers:                     params.OtelCol.Spec.HostUsers,
					HostAliases:                   params.OtelCol.Spec.HostAliases,
					ShareProcessNamespace:         &params.OtelCol.Spec.ShareProcessNamespace,
					ImagePullSecrets:              params.OtelCol.Spec.ImagePullSecrets,
					EnableServiceLinks:            params.OtelCol.Spec.EnableServiceLinks,
					Tolerations:                   params.OtelCol.Spec.Tolerations,
					NodeSelector:                  params.OtelCol.Spec.NodeSelector,
					SecurityContext:               params.OtelCol.Spec.PodSecurityContext,
					PriorityClassName:             params.OtelCol.Spec.PriorityClassName,
					Affinity:                      params.OtelCol.Spec.Affinity,
					TopologySpreadConstraints:     params.OtelCol.Spec.TopologySpreadConstraints,
					TerminationGracePeriodSeconds: params.OtelCol.Spec.TerminationGracePeriodSeconds,
				},
			},
			Replicas:                             manifestutils.GetDesiredReplicas(params.OtelCol),
			RevisionHistoryLimit:                 params.OtelCol.Spec.RevisionHistoryLimit,
			PodManagementPolicy:                  statefulSetPodManagementPolicy(params.OtelCol),
			VolumeClaimTemplates:                 VolumeClaimTemplates(params.OtelCol),
			PersistentVolumeClaimRetentionPolicy: params.OtelCol.Spec.PersistentVolumeClaimRetentionPolicy,
		},
	}, nil
}

// StatefulSetImmutableFields returns a StatefulSet that carries only the fields of the collector's
// StatefulSet that cannot change after creation, other than the selector.
func StatefulSetImmutableFields(otelcol v1beta1.OpenTelemetryCollector) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			ServiceName:          statefulSetServiceName(otelcol),
			PodManagementPolicy:  statefulSetPodManagementPolicy(otelcol),
			VolumeClaimTemplates: VolumeClaimTemplates(*otelcol.DeepCopy()),
		},
	}
}

func statefulSetServiceName(otelcol v1beta1.OpenTelemetryCollector) string {
	if otelcol.Spec.ServiceName == "" {
		return naming.HeadlessService(otelcol.Name)
	}
	return otelcol.Spec.ServiceName
}

func statefulSetPodManagementPolicy(otelcol v1beta1.OpenTelemetryCollector) appsv1.PodManagementPolicyType {
	if otelcol.Spec.PodManagementPolicy == "" {
		return appsv1.ParallelPodManagement
	}
	return otelcol.Spec.PodManagementPolicy
}
