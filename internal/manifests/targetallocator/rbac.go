// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func ClusterRole(params Params) *rbacv1.ClusterRole {
	ta := params.TargetAllocator
	name := naming.TAClusterRole(ta.Name, ta.Namespace)
	labels := manifestutils.Labels(ta.ObjectMeta, name, ta.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: ta.Annotations,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "nodes", "nodes/metrics", "services", "endpoints", "pods", "configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"monitoring.coreos.com"},
				Resources: []string{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				NonResourceURLs: []string{"/metrics"},
				Verbs:           []string{"get"},
			},
		},
	}
}

func ClusterRoleBinding(params Params) *rbacv1.ClusterRoleBinding {
	ta := params.TargetAllocator
	name := naming.TAClusterRoleBinding(ta.Name, ta.Namespace)
	labels := manifestutils.Labels(ta.ObjectMeta, name, ta.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: ta.Annotations,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ServiceAccountName(ta),
				Namespace: ta.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     naming.TAClusterRole(ta.Name, ta.Namespace),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}
