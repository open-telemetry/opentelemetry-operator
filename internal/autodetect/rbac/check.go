// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// CheckRBACPermissions checks if the operator has the needed permissions to
// create cluster-scoped RBAC resources (ClusterRole, ClusterRoleBinding).
func CheckRBACPermissions(ctx context.Context, reviewer *rbac.Reviewer) (admission.Warnings, error) {
	rules := []*rbacv1.PolicyRule{
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterrolebindings", "clusterroles"},
			Verbs:     []string{"create", "delete", "get", "list", "patch", "update"},
		},
	}
	return checkPermissions(ctx, reviewer, rules)
}

// CheckNamespacedRBACPermissions checks if the operator has the needed
// permissions to create namespace-scoped RBAC resources (Role, RoleBinding).
func CheckNamespacedRBACPermissions(ctx context.Context, reviewer *rbac.Reviewer) (admission.Warnings, error) {
	rules := []*rbacv1.PolicyRule{
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"rolebindings", "roles"},
			Verbs:     []string{"create", "delete", "get", "list", "patch", "update"},
		},
	}
	return checkPermissions(ctx, reviewer, rules)
}

func checkPermissions(ctx context.Context, reviewer *rbac.Reviewer, rules []*rbacv1.PolicyRule) (admission.Warnings, error) {
	namespace, err := autodetectutils.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	serviceAccount, err := autodetectutils.GetOperatorServiceAccount()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	if subjectAccessReviews, err := reviewer.CheckPolicyRules(ctx, serviceAccount, namespace, rules...); err != nil {
		return nil, fmt.Errorf("%s: %w", "unable to check rbac rules", err)
	} else if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return nil, nil
}
