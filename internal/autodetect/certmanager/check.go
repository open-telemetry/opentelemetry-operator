// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	rbac "github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// CheckCertManagerPermissions checks if the operator has the needed permissions to manage cert-manager certificates automatically.
// If the RBAC is there, no errors nor warnings are returned.
func CheckCertManagerPermissions(ctx context.Context, reviewer *rbac.Reviewer) (admission.Warnings, error) {
	namespace, err := autodetectutils.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	serviceAccount, err := autodetectutils.GetOperatorServiceAccount()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	rules := []*rbacv1.PolicyRule{
		{
			APIGroups: []string{"cert-manager.io"},
			Resources: []string{"issuers", "certificaterequests", "certificates"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		},
	}

	subjectAccessReviews, err := reviewer.CheckPolicyRules(ctx, serviceAccount, namespace, rules...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "unable to check rbac rules", err)
	}
	for _, review := range subjectAccessReviews {
		if !review.Status.Allowed && review.Status.EvaluationError != "" {
			return nil, fmt.Errorf("unable to evaluate cert-manager permissions: %s", review.Status.EvaluationError)
		}
	}
	if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return nil, nil
}
