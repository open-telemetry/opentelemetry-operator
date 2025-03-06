// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

type k8sobjectsConfig struct {
	Objects []k8sObject `yaml:"objects"`
}

type k8sObject struct {
	Name  string `yaml:"name"`
	Mode  string `yaml:"mode"`
	Group string `yaml:"group,omitempty"`
}

func generatek8sobjectsClusterRoleRules(_ logr.Logger, config k8sobjectsConfig) ([]rbacv1.PolicyRule, error) {
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sobjectsreceiver#rbac
	prs := []rbacv1.PolicyRule{}
	for _, obj := range config.Objects {
		permissions := []string{"list"}
		if obj.Mode == "pull" && (obj.Name != "events" && obj.Name != "events.k8s.io") {
			permissions = append(permissions, "get")
		} else if obj.Mode == "watch" {
			permissions = append(permissions, "watch")
		}
		prs = append(prs, rbacv1.PolicyRule{
			APIGroups: []string{obj.Group},
			Resources: []string{obj.Name},
			Verbs:     permissions,
		})
	}
	return prs, nil
}
