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

package processors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
)

func TestGenerateK8SAttrRbacRules(t *testing.T) {
	type args struct {
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []rbacv1.PolicyRule
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "default config with empty metadata",
			args: args{
				config: map[string]interface{}{
					"extract": map[string]interface{}{
						"metadata":    []string{},
						"labels":      []interface{}{},
						"annotations": []interface{}{},
					},
				},
			},
			want: []rbacv1.PolicyRule{
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
			wantErr: assert.NoError,
		},
		{
			name: "config with deployment metadata",
			args: args{
				config: map[string]interface{}{
					"extract": map[string]interface{}{
						"metadata":    []string{"k8s.deployment.uid", "k8s.deployment.name"},
						"labels":      []interface{}{},
						"annotations": []interface{}{},
					},
				},
			},
			want: []rbacv1.PolicyRule{
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
			wantErr: assert.NoError,
		},
		{
			name: "config with node metadata",
			args: args{
				config: map[string]interface{}{
					"extract": map[string]interface{}{
						"metadata":    []string{"k8s.node.name"},
						"labels":      []interface{}{},
						"annotations": []interface{}{},
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "namespaces"},
					Verbs:     []string{"get", "watch", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid config",
			args: args{
				config: "hi",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "config with invalid metadata",
			args: args{
				config: map[string]interface{}{
					"extract": map[string]interface{}{
						"metadata":    []string{"invalid.metadata"},
						"labels":      []interface{}{},
						"annotations": []interface{}{},
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "namespaces"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := processors.ProcessorFor("k8sattributes")
			got, err := parser.GetClusterRoleRules(logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetClusterRoleRules(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetClusterRoleRules(%v)", tt.args.config)
		})
	}
}
