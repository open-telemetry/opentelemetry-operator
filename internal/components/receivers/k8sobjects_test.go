// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Test_generatek8sobjectsRbacRules(t *testing.T) {
	tests := []struct {
		name   string
		config k8sobjectsConfig
		want   []rbacv1.PolicyRule
	}{
		{
			name: "basic watch mode",
			config: k8sobjectsConfig{
				Objects: []k8sObject{
					{
						Name:  "pods",
						Mode:  "watch",
						Group: "v1",
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"v1"},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			name: "pull mode with events",
			config: k8sobjectsConfig{
				Objects: []k8sObject{
					{
						Name:  "events",
						Mode:  "pull",
						Group: "v1",
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"v1"},
					Resources: []string{"events"},
					Verbs:     []string{"list"},
				},
			},
		},
		{
			name: "pull mode with non-events",
			config: k8sobjectsConfig{
				Objects: []k8sObject{
					{
						Name:  "pods",
						Mode:  "pull",
						Group: "v1",
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"v1"},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "get"},
				},
			},
		},
		{
			name: "multiple objects",
			config: k8sobjectsConfig{
				Objects: []k8sObject{
					{
						Name:  "pods",
						Mode:  "pull",
						Group: "v1",
					},
					{
						Name:  "events",
						Mode:  "pull",
						Group: "v1",
					},
					{
						Name:  "deployments",
						Mode:  "watch",
						Group: "apps/v1",
					},
				},
			},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"v1"},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "get"},
				},
				{
					APIGroups: []string{"v1"},
					Resources: []string{"events"},
					Verbs:     []string{"list"},
				},
				{
					APIGroups: []string{"apps/v1"},
					Resources: []string{"deployments"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatek8sobjectsClusterRoleRules(logr.Logger{}, tt.config)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
