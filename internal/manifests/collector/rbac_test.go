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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"

	_ "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/processor"
)

func TestDesiredClusterRoles(t *testing.T) {

	// No Cluster Roles
	params, err := newParams("", "testdata/prometheus-exporter.yaml")
	assert.NoError(t, err, "No")

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
	}

	for _, test := range tests {
		params, err := newParams("", test.configPath)
		assert.NoError(t, err, test.desc)

		cr, err := ClusterRole(params)
		require.NoError(t, err)
		assert.Equal(t, test.expectedRules, cr.Rules, test.desc)
	}
}

func TestDesiredClusterRolBinding(t *testing.T) {

	// No ClusterRoleBinding
	params, err := newParams("", "testdata/prometheus-exporter.yaml")
	assert.NoError(t, err)

	crb, err := ClusterRoleBinding(params)
	require.NoError(t, err)
	assert.Nil(t, crb)

	// Create ClusterRoleBinding
	params, err = newParams("", "testdata/rbac_resourcedetectionprocessor_k8s.yaml")
	assert.NoError(t, err)

	crb, err = ClusterRoleBinding(params)
	require.NoError(t, err)
	assert.NotNil(t, crb)
}
