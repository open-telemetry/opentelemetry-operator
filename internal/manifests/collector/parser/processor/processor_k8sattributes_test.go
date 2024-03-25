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

package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestK8sAttributesRBAC(t *testing.T) {

	tests := []struct {
		name          string
		config        map[interface{}]interface{}
		expectedRules []rbacv1.PolicyRule
	}{
		{
			name:   "no extra parameters",
			config: nil,
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "namespaces"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
		{
			name: "extract k8s.deployment.uid",
			config: map[interface{}]interface{}{
				"extract": map[interface{}]interface{}{
					"metadata": []interface{}{
						"k8s.deployment.uid",
					},
				},
			},
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
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"replicasets"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
		{
			name: "extract k8s.deployment.name",
			config: map[interface{}]interface{}{
				"extract": map[interface{}]interface{}{
					"metadata": []interface{}{
						"k8s.deployment.name",
					},
				},
			},
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
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"replicasets"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewK8sAttributesParser(logger, "test", tt.config)
			rules := p.GetRBACRules()
			assert.Equal(t, tt.expectedRules, rules)
		})

	}

}
