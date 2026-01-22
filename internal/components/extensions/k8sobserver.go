// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

type k8sobserverConfig struct {
	ObservePods     bool `mapstructure:"observe_pods"`
	ObserveServices bool `mapstructure:"observe_services"`
	ObserveNodes    bool `mapstructure:"observe_nodes"`
}

func generatek8sobserverRbacRules(_ logr.Logger, config k8sobserverConfig) ([]rbacv1.PolicyRule, error) {
	prs := []rbacv1.PolicyRule{}

	if config.ObservePods {
		prs = append(prs, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"list", "watch"},
		})
	}

	if config.ObserveServices {
		prs = append(prs, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"services"},
			Verbs:     []string{"list", "watch"},
		})
	}

	if config.ObserveNodes {
		prs = append(prs, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"list", "watch"},
		})
	}

	return prs, nil
}
