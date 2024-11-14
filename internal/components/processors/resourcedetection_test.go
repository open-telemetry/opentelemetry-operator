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

func TestGenerateResourceDetectionRbacRules(t *testing.T) {
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []rbacv1.PolicyRule
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "default config with no detectors",
			args: args{
				config: map[string]interface{}{
					"detectors": []string{},
				},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "config with k8snode detector",
			args: args{
				config: map[string]interface{}{
					"detectors": []string{"k8snode"},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "config with openshift detector",
			args: args{
				config: map[string]interface{}{
					"detectors": []string{"openshift"},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"config.openshift.io"},
					Resources: []string{"infrastructures", "infrastructures/status"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "config with multiple detectors",
			args: args{
				config: map[string]interface{}{
					"detectors": []string{"k8snode", "openshift"},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{"config.openshift.io"},
					Resources: []string{"infrastructures", "infrastructures/status"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "config with invalid detector",
			args: args{
				config: map[string]interface{}{
					"detectors": []string{"invalid"},
				},
			},
			want:    nil,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := processors.ProcessorFor("resourcedetection")
			got, err := parser.GetClusterRoleRules(logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetClusterRoleRules(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetClusterRoleRules(%v)", tt.args.config)
		})
	}
}
