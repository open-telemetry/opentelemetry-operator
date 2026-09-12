// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
)

func TestGenerateK8SAttrRbacRules(t *testing.T) {
	type args struct {
		config any
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
				config: map[string]any{
					"extract": map[string]any{
						"metadata":    []string{},
						"labels":      []any{},
						"annotations": []any{},
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
				config: map[string]any{
					"extract": map[string]any{
						"metadata":    []string{"k8s.deployment.uid", "k8s.deployment.name"},
						"labels":      []any{},
						"annotations": []any{},
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
				config: map[string]any{
					"extract": map[string]any{
						"metadata":    []string{"k8s.node.name"},
						"labels":      []any{},
						"annotations": []any{},
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
				config: map[string]any{
					"extract": map[string]any{
						"metadata":    []string{"invalid.metadata"},
						"labels":      []any{},
						"annotations": []any{},
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
		{
			name: "config with service.name metadata",
			args: args{
				config: map[string]any{
					"extract": map[string]any{
						"metadata":    []string{"service.name"},
						"labels":      []any{},
						"annotations": []any{},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := processors.ProcessorFor("k8sattributes")
			got, err := parser.GetRBACRules(logger, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRBACRules(%v)", tt.args.config)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetRBACRules(%v)", tt.args.config)
		})
	}
}

func TestGenerateK8SAttrRbacRulesWithFilterNamespace(t *testing.T) {
	tests := []struct {
		name             string
		config           any
		wantClusterRules []rbacv1.PolicyRule
		wantNsRules      map[string][]rbacv1.PolicyRule
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "filter.namespace set - no cluster rules, namespace-scoped rules instead",
			config: map[string]any{
				"filter": map[string]any{
					"namespace": "my-namespace",
				},
				"extract": map[string]any{
					"metadata":    []string{},
					"labels":      []any{},
					"annotations": []any{},
				},
			},
			wantClusterRules: nil,
			wantNsRules: map[string][]rbacv1.PolicyRule{
				"my-namespace": {
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
			wantErr: assert.NoError,
		},
		{
			name: "filter.namespace with node metadata",
			config: map[string]any{
				"filter": map[string]any{
					"namespace": "test-ns",
				},
				"extract": map[string]any{
					"metadata":    []string{"k8s.node.name"},
					"labels":      []any{},
					"annotations": []any{},
				},
			},
			wantClusterRules: nil,
			wantNsRules: map[string][]rbacv1.PolicyRule{
				"test-ns": {
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
			},
			wantErr: assert.NoError,
		},
		{
			name: "no filter.namespace - cluster rules, no namespace-scoped rules",
			config: map[string]any{
				"extract": map[string]any{
					"metadata":    []string{},
					"labels":      []any{},
					"annotations": []any{},
				},
			},
			wantClusterRules: []rbacv1.PolicyRule{
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
			wantNsRules: nil,
			wantErr:     assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := processors.ProcessorFor("k8sattributes")
			clusterRules, err := parser.GetRBACRules(logger, tt.config)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.wantClusterRules, clusterRules, "GetRBACRules (cluster-scoped)")

			provider, ok := parser.(components.NamespacedRBACRuleProvider)
			assert.True(t, ok, "parser should implement NamespacedRBACRuleProvider")
			nsRules, err := provider.GetNamespacedRBACRules(logger, tt.config)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.wantNsRules, nsRules, "GetNamespacedRBACRules")
		})
	}
}
