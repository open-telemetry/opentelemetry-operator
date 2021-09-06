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

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/version"
	"github.com/signalfx/splunk-otel-operator/pkg/collector/upgrade"
)

func TestHealthCheckEndpointMigration(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.SplunkOtelAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "splunk-otel-operator",
			},
		},
		Spec: v1alpha1.SplunkOtelAgentSpec{
			Config: `extensions:
  health_check:
  health_check/1: ""
  health_check/2:
    endpoint: "localhost:13133"
  health_check/3:
    port: 13133
`,
		},
	}
	existing.Status.Version = "0.23.0"

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, `extensions:
  health_check: null
  health_check/1: ""
  health_check/2:
    endpoint: localhost:13133
  health_check/3:
    endpoint: 0.0.0.0:13133
`, res.Spec.Config)
	assert.Equal(t, "upgrade to v0.24.0 migrated the property 'port' to 'endpoint' for extension \"health_check/3\"", res.Status.Messages[0])
}
