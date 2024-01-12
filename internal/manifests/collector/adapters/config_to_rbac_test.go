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

package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestConfigRBAC(t *testing.T) {
	tests := []struct {
		desc          string
		config        string
		expectedRules []rbacv1.PolicyRule
	}{
		{
			desc: "No processors",
			config: `processors:
service:
  traces:
    processors:`,
			expectedRules: ([]rbacv1.PolicyRule)(nil),
		},
		{
			desc: "processors no rbac",
			config: `processors:
  batch:
service:
  pipelines:
    traces:
      processors: [batch]`,
			expectedRules: ([]rbacv1.PolicyRule)(nil),
		},
		{
			desc: "resourcedetection-processor k8s",
			config: `processors:
  resourcedetection:
    detectors: [kubernetes]
service:
  pipelines:
    traces:
      processors: [resourcedetection]`,
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		{
			desc: "resourcedetection-processor openshift",
			config: `processors:
  resourcedetection:
    detectors: [openshift]
service:
  pipelines:
    traces:
      processors: [resourcedetection]`,
			expectedRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"config.openshift.io"},
					Resources: []string{"infrastructures", "infrastructures/status"},
					Verbs:     []string{"get", "watch", "list"},
				},
			},
		},
	}

	var logger = logf.Log.WithName("collector-unit-tests")

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			config, err := ConfigFromString(tt.config)
			require.NoError(t, err, tt.desc)
			require.NotEmpty(t, config, tt.desc)

			// test
			rules := ConfigToRBAC(logger, config)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRules, rules, tt.desc)
		})
	}
}
