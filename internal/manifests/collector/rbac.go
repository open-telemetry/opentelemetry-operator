// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"github.com/samber/lo"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func ClusterRole(params manifests.Params) (*rbacv1.ClusterRole, error) {
	confStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	configFromString, err := adapters.ConfigFromString(confStr)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil, nil
	}
	rules := lo.Filter(adapters.ConfigToRBAC(params.Log, configFromString), func(policy authz.DynamicRolePolicy, _ int) bool {
		return isClusterRolePolicy(policy)
	})
	if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: params.OtelCol.Annotations,
			Labels:      labels,
		},
		Rules: lo.FlatMap(rules, func(item authz.DynamicRolePolicy, _ int) []rbacv1.PolicyRule {
			return item.Rules
		}),
	}, nil
}

func ClusterRoleBinding(params manifests.Params) (*rbacv1.ClusterRoleBinding, error) {
	confStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}
	configFromString, err := adapters.ConfigFromString(confStr)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil, nil
	}
	rules := lo.Filter(adapters.ConfigToRBAC(params.Log, configFromString), func(policy authz.DynamicRolePolicy, _ int) bool {
		return isClusterRolePolicy(policy)
	})
	if len(rules) == 0 {
		return nil, nil
	}

	name := naming.ClusterRoleBinding(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: params.OtelCol.Annotations,
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

func Role(params manifests.Params) ([]*rbacv1.Role, error) {
	confStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	configFromString, err := adapters.ConfigFromString(confStr)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil, nil
	}
	rules := lo.Filter(adapters.ConfigToRBAC(params.Log, configFromString), func(policy authz.DynamicRolePolicy, _ int) bool {
		return !isClusterRolePolicy(policy)
	})
	if len(rules) == 0 {
		return nil, nil
	}

	name := naming.Role(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	var roles []*rbacv1.Role
	for _, rule := range rules {
		for _, ns := range rule.Namespaces {
			roles = append(roles, &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   ns,
					Annotations: params.OtelCol.Annotations,
					Labels:      labels,
				},
				Rules: rule.Rules,
			})
		}
	}
	return roles, nil
}

func RoleBinding(params manifests.Params) ([]*rbacv1.RoleBinding, error) {
	confStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	configFromString, err := adapters.ConfigFromString(confStr)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil, nil
	}
	rules := lo.Filter(adapters.ConfigToRBAC(params.Log, configFromString), func(policy authz.DynamicRolePolicy, _ int) bool {
		return !isClusterRolePolicy(policy)
	})
	if len(rules) == 0 {
		return nil, nil
	}

	name := naming.RoleBinding(params.OtelCol.Name, params.OtelCol.Namespace)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())

	var roles []*rbacv1.RoleBinding
	for _, rule := range rules {
		for _, ns := range rule.Namespaces {
			roles = append(roles, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   ns,
					Annotations: params.OtelCol.Annotations,
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
					Name:     naming.Role(params.OtelCol.Name, params.OtelCol.Namespace),
					APIGroup: "rbac.authorization.k8s.io",
				},
			})
		}
	}
	return roles, nil
}

func isClusterRolePolicy(policy authz.DynamicRolePolicy) bool {
	// An empty namespace list indicates that the policy rules belong to ClusterRole resources.
	return len(policy.Namespaces) == 0
}
