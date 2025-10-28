// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestGeneratek8sobserverRbacRules(t *testing.T) {
	tests := []struct {
		name   string
		config k8sobserverConfig
		want   []rbacv1.PolicyRule
	}{
		{
			name:   "none enabled",
			config: k8sobserverConfig{},
			want:   []rbacv1.PolicyRule{},
		},
		{
			name:   "pods only",
			config: k8sobserverConfig{ObservePods: true},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			name:   "services only",
			config: k8sobserverConfig{ObserveServices: true},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			name:   "nodes only",
			config: k8sobserverConfig{ObserveNodes: true},
			want: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"list", "watch"},
				},
			},
		},
		{
			name:   "pods and services",
			config: k8sobserverConfig{ObservePods: true, ObserveServices: true},
			want: []rbacv1.PolicyRule{
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
			},
		},
		{
			name:   "all enabled",
			config: k8sobserverConfig{ObservePods: true, ObserveServices: true, ObserveNodes: true},
			want: []rbacv1.PolicyRule{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatek8sobserverRbacRules(logr.Logger{}, tt.config)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
