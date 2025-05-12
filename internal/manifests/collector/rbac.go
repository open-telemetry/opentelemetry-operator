// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func ClusterRole(params manifests.Params) (*rbacv1.ClusterRole, error) {
	rules, err := params.OtelCol.Spec.Config.GetAllRbacRules(params.Log)
	if err != nil {
		return nil, err
	} else if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Rules: rules,
	}, nil
}

func ClusterRoleBinding(params manifests.Params) (*rbacv1.ClusterRoleBinding, error) {
	rules, err := params.OtelCol.Spec.Config.GetAllRbacRules(params.Log)
	if err != nil {
		return nil, err
	} else if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRoleBinding(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ServiceAccountName(params.OtelCol),
				Namespace: params.OtelCol.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}, nil
}

func CheckRbacRules(params manifests.Params, saName string) ([]string, error) {
	ctx := context.Background()

	rules, err := params.OtelCol.Spec.Config.GetAllRbacRules(params.Log)
	if err != nil {
		return nil, err
	}

	r := []*rbacv1.PolicyRule{}

	for _, rule := range rules {
		rule := rule
		r = append(r, &rule)
	}

	if subjectAccessReviews, err := params.Reviewer.CheckPolicyRules(ctx, saName, params.OtelCol.Namespace, r...); err != nil {
		return nil, fmt.Errorf("%s: %w", "unable to check rbac rules", err)
	} else if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return nil, nil
}
