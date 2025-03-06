// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

type k8sclusterConfig struct {
	Distribution string `mapstructure:"distribution"`
}

func generatek8sclusterRbacRules(_ logr.Logger, cfg k8sclusterConfig) ([]rbacv1.PolicyRule, error) {
	policyRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"events",
				"namespaces",
				"namespaces/status",
				"nodes",
				"nodes/spec",
				"pods",
				"pods/status",
				"replicationcontrollers",
				"replicationcontrollers/status",
				"resourcequotas",
				"services",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{
				"daemonsets",
				"deployments",
				"replicasets",
				"statefulsets",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"extensions"},
			Resources: []string{
				"daemonsets",
				"deployments",
				"replicasets",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"batch"},
			Resources: []string{
				"jobs",
				"cronjobs",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"autoscaling"},
			Resources: []string{"horizontalpodautoscalers"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	if cfg.Distribution == "openshift" {
		policyRules = append(policyRules, rbacv1.PolicyRule{
			APIGroups: []string{"quota.openshift.io"},
			Resources: []string{"clusterresourcequotas"},
			Verbs:     []string{"get", "list", "watch"},
		})
	}
	return policyRules, nil
}
