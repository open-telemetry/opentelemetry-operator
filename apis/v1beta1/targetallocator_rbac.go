// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// targetAllocatorCRPolicyRules are the policy rules required for the CR functionality.
// These rules are expected to be granted via a ClusterRole.
var targetAllocatorCRPolicyRules = []*rbacv1.PolicyRule{
	{
		APIGroups: []string{"monitoring.coreos.com"},
		Resources: []string{"servicemonitors", "podmonitors"},
		Verbs:     []string{"*"},
	}, {
		APIGroups: []string{""},
		Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods", "namespaces"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}, {
		APIGroups: []string{"discovery.k8s.io"},
		Resources: []string{"endpointslices"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{"networking.k8s.io"},
		Resources: []string{"ingresses"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		NonResourceURLs: []string{"/metrics"},
		Verbs:           []string{"get"},
	}, {
		NonResourceURLs: []string{"/api", "/api/*", "/apis", "/apis/*"},
		Verbs:           []string{"get"},
	},
}

// targetAllocatorCRNamespacedPolicyRules are the policy rules that should be granted
// via a Role (namespace-scoped) rather than a ClusterRole. Secrets are namespace-scoped
// and should not be granted cluster-wide access.
var targetAllocatorCRNamespacedPolicyRules = []*rbacv1.PolicyRule{
	{
		APIGroups: []string{""},
		Resources: []string{"secrets"},
		Verbs:     []string{"get", "list", "watch"},
	},
}

func CheckTargetAllocatorPrometheusCRPolicyRules(
	ctx context.Context,
	reviewer *rbac.Reviewer,
	namespace string,
	serviceAccountName string,
) (warnings []string, err error) {
	// Check cluster-scoped rules (ClusterRole)
	subjectAccessReviews, err := reviewer.CheckPolicyRules(
		ctx,
		serviceAccountName,
		namespace,
		targetAllocatorCRPolicyRules...,
	)
	if err != nil {
		return []string{}, fmt.Errorf("unable to check rbac rules %w", err)
	}

	// Check namespace-scoped rules (Role) — secrets should be scoped to the namespace
	namespacedReviews, err := reviewer.CheckPolicyRules(
		ctx,
		serviceAccountName,
		namespace,
		targetAllocatorCRNamespacedPolicyRules...,
	)
	if err != nil {
		return []string{}, fmt.Errorf("unable to check namespace-scoped rbac rules %w", err)
	}
	subjectAccessReviews = append(subjectAccessReviews, namespacedReviews...)

	if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return []string{}, nil
}
