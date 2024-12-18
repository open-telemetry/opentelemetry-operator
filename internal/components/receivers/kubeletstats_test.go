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
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestGenerateKubeletStatsRbacRules(t *testing.T) {
	baseRule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"nodes/stats"},
		Verbs:     []string{"get"},
	}

	proxyRule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"nodes/proxy"},
		Verbs:     []string{"get"},
	}

	tests := []struct {
		name           string
		config         kubeletStatsConfig
		expectedRules  []rbacv1.PolicyRule
		expectedErrMsg string
	}{
		{
			name:          "Default config",
			config:        kubeletStatsConfig{},
			expectedRules: []rbacv1.PolicyRule{baseRule},
		},
		{
			name: "Extra metadata labels",
			config: kubeletStatsConfig{
				ExtraMetadataLabels: []string{"label1", "label2"},
			},
			expectedRules: []rbacv1.PolicyRule{baseRule, proxyRule},
		},
		{
			name: "CPU limit utilization enabled",
			config: kubeletStatsConfig{
				Metrics: metrics{
					K8sContainerCPULimitUtilization: metricConfig{Enabled: true},
				},
			},
			expectedRules: []rbacv1.PolicyRule{baseRule, proxyRule},
		},
		{
			name: "Memory request utilization enabled",
			config: kubeletStatsConfig{
				Metrics: metrics{
					K8sPodMemoryRequestUtilization: metricConfig{Enabled: true},
				},
			},
			expectedRules: []rbacv1.PolicyRule{baseRule, proxyRule},
		},
		{
			name: "No extra permissions needed",
			config: kubeletStatsConfig{
				Metrics: metrics{
					K8sContainerCPULimitUtilization: metricConfig{Enabled: false},
				},
			},
			expectedRules: []rbacv1.PolicyRule{baseRule},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := generateKubeletStatsClusterRoleRules(logr.Logger{}, tt.config)

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRules, rules)
			}
		})
	}
}
