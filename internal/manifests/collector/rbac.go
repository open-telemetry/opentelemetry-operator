// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func ClusterRole(params manifests.Params) (*rbacv1.ClusterRole, error) {
	rules, err := params.OtelCol.Spec.Config.GetAllClusterRoleRbacRules(params.Log)
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
	rules, err := params.OtelCol.Spec.Config.GetAllClusterRoleRbacRules(params.Log)
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

func Role(params manifests.Params) ([]client.Object, error) {
	roles, err := params.OtelCol.Spec.Config.GetAllRbacRoles(params.Log, params.OtelCol.Name)
	if err != nil {
		return nil, err
	}

	name := naming.Role(params.OtelCol.Name, params.OtelCol.Namespace)

	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	// Convert []*rbacv1.Role to []client.Object
	result := make([]client.Object, len(roles))
	for i, role := range roles {
		role.ObjectMeta.Labels = labels
		role.ObjectMeta.Annotations = annotations
		result[i] = role
	}

	return result, nil
}

func RoleBinding(params manifests.Params) ([]client.Object, error) {
	rbs, err := params.OtelCol.Spec.Config.GetAllRbacRoleBindings(params.Log, ServiceAccountName(params.OtelCol), params.OtelCol.Name, params.OtelCol.Namespace)
	if err != nil {
		return nil, err
	} else if len(rbs) == 0 {
		return nil, nil
	}

	name := naming.RoleBinding(params.OtelCol.Name, params.OtelCol.Namespace)

	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	// Convert []*rbacv1.RoleBinding to []client.Object
	result := make([]client.Object, len(rbs))
	for i, rb := range rbs {
		rb.ObjectMeta.Labels = labels
		rb.ObjectMeta.Annotations = annotations
		result[i] = rb
	}

	return result, nil
}

func CheckRbacRules(params manifests.Params, saName string) ([]string, error) {
	ctx := context.Background()

	rules, err := params.OtelCol.Spec.Config.GetAllClusterRoleRbacRules(params.Log)
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
