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
	"github.com/open-telemetry/opentelemetry-operator/internal/otelconfig"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func ClusterRole(params manifests.Params) (*rbacv1.ClusterRole, error) {
	rules, err := otelconfig.GetAllRbacRules(&params.OtelCol.Spec.Config, params.Log)
	if err != nil {
		return nil, err
	} else if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
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
	rules, err := otelconfig.GetAllRbacRules(&params.OtelCol.Spec.Config, params.Log)
	if err != nil {
		return nil, err
	} else if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRoleBinding(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
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

// Roles returns namespace-scoped Roles for components that require namespace-scoped
// RBAC (e.g. k8sattributes processor with filter.namespace set).
func Roles(params manifests.Params) ([]*rbacv1.Role, error) {
	nsRules, err := otelconfig.GetAllNamespacedRbacRules(&params.OtelCol.Spec.Config, params.Log)
	if err != nil {
		return nil, err
	}
	if len(nsRules) == 0 {
		return nil, nil
	}

	var roles []*rbacv1.Role
	for ns, rules := range nsRules {
		name := naming.Role(params.OtelCol.Name, ns)
		labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

		annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
		if err != nil {
			return nil, err
		}

		roles = append(roles, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   ns,
				Annotations: annotations,
				Labels:      labels,
			},
			Rules: rules,
		})
	}
	return roles, nil
}

// RoleBindings returns namespace-scoped RoleBindings for components that require
// namespace-scoped RBAC.
func RoleBindings(params manifests.Params) ([]*rbacv1.RoleBinding, error) {
	nsRules, err := otelconfig.GetAllNamespacedRbacRules(&params.OtelCol.Spec.Config, params.Log)
	if err != nil {
		return nil, err
	}
	if len(nsRules) == 0 {
		return nil, nil
	}

	var bindings []*rbacv1.RoleBinding
	for ns := range nsRules {
		name := naming.RoleBinding(params.OtelCol.Name, ns)
		labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

		annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
		if err != nil {
			return nil, err
		}

		bindings = append(bindings, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   ns,
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
				Kind:     "Role",
				Name:     naming.Role(params.OtelCol.Name, ns),
				APIGroup: "rbac.authorization.k8s.io",
			},
		})
	}
	return bindings, nil
}

func CheckRbacRules(params manifests.Params, saName string) ([]string, error) {
	ctx := context.Background()

	rules, err := otelconfig.GetAllRbacRules(&params.OtelCol.Spec.Config, params.Log)
	if err != nil {
		return nil, err
	}

	r := []*rbacv1.PolicyRule{}

	for _, rule := range rules {
		r = append(r, &rule)
	}

	if subjectAccessReviews, err := params.Reviewer.CheckPolicyRules(ctx, saName, params.OtelCol.Namespace, r...); err != nil {
		return nil, fmt.Errorf("%s: %w", "unable to check rbac rules", err)
	} else if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return nil, nil
}
