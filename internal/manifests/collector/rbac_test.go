// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestDesiredClusterRoles(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", nil)
	assert.NoError(t, err)

	cr, err := ClusterRole(params)
	require.NoError(t, err)
	assert.Nil(t, cr)

	tests := []struct {
		desc          string
		configPath    string
		expectedRules []rbacv1.PolicyRule
	}{
		{
			desc:       "resourcedetection processor - kubernetes detector",
			configPath: "testdata/rbac_resourcedetectionprocessor_k8s.yaml",
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		{
			desc:       "resourcedetection processor - openshift detector",
			configPath: "testdata/rbac_resourcedetectionprocessor_openshift.yaml",
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"config.openshift.io"},
					Resources: []string{"infrastructures", "infrastructures/status"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
		{
			desc:       "k8s_observer extension - all resources enabled",
			configPath: "testdata/rbac_k8sobserver_extension.yaml",
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     []string{"list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			desc:       "k8s_observer extension - only pods enabled",
			configPath: "testdata/rbac_k8sobserver_partial.yaml",
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			desc:       "k8sattributes processor - service.name metadata",
			configPath: "testdata/rbac_k8sattributes_service_name.yaml",
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "namespaces"},
					Verbs:     []string{"get", "watch", "list"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"replicasets"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
	}

	for _, test := range tests {
		params, err := newParams("", test.configPath, nil)
		assert.NoError(t, err, test.desc)

		cr, err := ClusterRole(params)
		require.NoError(t, err)
		assert.Equal(t, test.expectedRules, cr.Rules, test.desc)
	}
}

func TestDesiredClusterRolBinding(t *testing.T) {

	// No ClusterRoleBinding
	params, err := newParams("", "testdata/prometheus-exporter.yaml", nil)
	assert.NoError(t, err)

	crb, err := ClusterRoleBinding(params)
	require.NoError(t, err)
	assert.Nil(t, crb)

	// Create ClusterRoleBinding
	params, err = newParams("", "testdata/rbac_resourcedetectionprocessor_k8s.yaml", nil)
	assert.NoError(t, err)

	crb, err = ClusterRoleBinding(params)
	require.NoError(t, err)
	assert.NotNil(t, crb)
}

func TestSkipRBACWhenCustomServiceAccount(t *testing.T) {
	// Test that ClusterRole and ClusterRoleBinding are not created when custom ServiceAccount is provided
	params, err := newParams("", "testdata/rbac_resourcedetectionprocessor_k8s.yaml", nil)
	assert.NoError(t, err)

	// Set custom ServiceAccount
	params.OtelCol.Spec.ServiceAccount = "my-custom-service-account"

	// Should return nil for ClusterRole when custom SA is provided
	cr, err := ClusterRole(params)
	require.NoError(t, err)
	assert.Nil(t, cr, "ClusterRole should not be created when custom ServiceAccount is provided")

	// Should return nil for ClusterRoleBinding when custom SA is provided
	crb, err := ClusterRoleBinding(params)
	require.NoError(t, err)
	assert.Nil(t, crb, "ClusterRoleBinding should not be created when custom ServiceAccount is provided")
}
