// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestK8sLeaderElectorRBACRules(t *testing.T) {
	rules, err := generatek8sleaderelectorRbacRules(logr.Discard(), nil)
	require.NoError(t, err)

	require.Len(t, rules, 1, "should return exactly one policy rule")

	rule := rules[0]

	assert.Equal(t, []string{"coordination.k8s.io"}, rule.APIGroups,
		"should target the coordination.k8s.io API group")
	assert.Equal(t, []string{"leases"}, rule.Resources,
		"should target the leases resource")
	assert.Equal(t,
		[]string{"get", "list", "watch", "create", "update", "patch", "delete"},
		rule.Verbs,
		"should include all required verbs for leader election",
	)
}

func TestK8sLeaderElectorRBACRulesMatchUpstreamDocs(t *testing.T) {
	// Validates the RBAC rules exactly match the upstream k8s_leader_elector
	// extension documentation:
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/k8sleaderelector
	rules, err := generatek8sleaderelectorRbacRules(logr.Discard(), nil)
	require.NoError(t, err)

	expected := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
	}

	assert.Equal(t, expected, rules)
}
