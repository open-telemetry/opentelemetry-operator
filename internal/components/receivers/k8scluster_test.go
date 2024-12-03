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

package receivers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Test_generatek8sclusterRbacRules(t *testing.T) {
	tests := []struct {
		name    string
		cfg     k8sclusterConfig
		want    []rbacv1.PolicyRule
		wantErr bool
	}{
		{
			name: "default configuration",
			cfg:  k8sclusterConfig{},
			want: []rbacv1.PolicyRule{
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
			},
			wantErr: false,
		},
		{
			name: "openshift configuration",
			cfg: k8sclusterConfig{
				Distribution: "openshift",
			},
			want: []rbacv1.PolicyRule{
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
				{
					APIGroups: []string{"quota.openshift.io"},
					Resources: []string{"clusterresourcequotas"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatek8sclusterRbacRules(logr.Discard(), tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
