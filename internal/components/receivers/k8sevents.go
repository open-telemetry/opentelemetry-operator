// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

type k8seventsConfig struct{}

func generatek8seventsRbacRules(_ logr.Logger, _ k8seventsConfig) ([]rbacv1.PolicyRule, error) {
	// The k8s Events Receiver needs the get permissions on the following resources always.
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"events",
			},
			Verbs: []string{"get", "list", "watch"},
		},
	}, nil
}
