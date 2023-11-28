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
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ClusterRole(params manifests.Params) *rbacv1.ClusterRole {
	configFromString, err := adapters.ConfigFromString(params.OtelCol.Spec.Config)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil
	}
	rules := adapters.ConfigToRBAC(params.Log, configFromString)

	if len(rules) == 0 {
		return nil
	}

	return &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace),
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "opentelemetry-collector",
				},
			},
			Rules: rules,
	}
}

func ClusterRoleBinding(params manifests.Params) *rbacv1.ClusterRoleBinding {
	configFromString, err := adapters.ConfigFromString(params.OtelCol.Spec.Config)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil
	}
	rules := adapters.ConfigToRBAC(params.Log, configFromString)

	if len(rules) == 0 {
		return nil
	}

	return &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.ClusterRoleBinding(params.OtelCol.Name),
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.ClusterRoleBinding(params.OtelCol.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "opentelemetry-collector",
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "ServiceAccount",
					Name: ServiceAccountName(params.OtelCol),
					Namespace: params.OtelCol.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
					Kind: "ClusterRole",
					Name: naming.ClusterRole(params.OtelCol.Name, params.OtelCol.Namespace),
					APIGroup: "rbac.authorization.k8s.io",
				},
		}
}
