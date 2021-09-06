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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/version"
	"github.com/signalfx/splunk-otel-operator/pkg/collector/adapters"
	"github.com/signalfx/splunk-otel-operator/pkg/collector/upgrade"
)

func TestRemoveQueuedRetryProcessor(t *testing.T) {
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
			Config: `processors:
  queued_retry:
  otherprocessor:
  queued_retry/second:
    compression: "on"
    reconnection_delay: 15
    num_workers: 123`,
		},
	}
	existing.Status.Version = "0.18.0"

	// sanity check
	require.Contains(t, existing.Spec.Config, "queued_retry")
	require.Contains(t, existing.Spec.Config, "queued_retry/second")
	require.Contains(t, existing.Spec.Config, "num_workers: 123") // checking one property is sufficient

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.NotContains(t, res.Spec.Config, "queued_retry:")
	assert.Contains(t, res.Spec.Config, "otherprocessor:")
	assert.NotContains(t, res.Spec.Config, "queued_retry/second:")
	assert.NotContains(t, res.Spec.Config, "num_workers: 123") // checking one property is sufficient
	assert.Contains(t, res.Status.Messages[0], "upgrade to v0.19.0 removed the processor")
}

func TestMigrateResourceType(t *testing.T) {
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
			Config: `processors:
  resource:
    type: some-type
`,
		},
	}
	existing.Status.Version = "0.18.0"

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, `processors:
  resource:
    attributes:
    - action: upsert
      key: opencensus.type
      value: some-type
`, res.Spec.Config)
	assert.Contains(t, res.Status.Messages[0], "upgrade to v0.19.0 migrated the property 'type' for processor")
}

func TestMigrateLabels(t *testing.T) {
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
			Config: `processors:
  resource:
    labels:
      cloud.zone: zone-1
      host.name: k8s-node
`,
		},
	}
	existing.Status.Version = "0.18.0"

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	actual, err := adapters.ConfigFromString(res.Spec.Config)
	require.NoError(t, err)
	actualProcessors := actual["processors"].(map[interface{}]interface{})
	actualProcessor := actualProcessors["resource"].(map[interface{}]interface{})
	actualAttrs := actualProcessor["attributes"].([]interface{})

	// verify
	assert.Len(t, actualAttrs, 2)
	assert.Nil(t, actualProcessor["labels"])
	assert.Contains(t, res.Status.Messages[0], "upgrade to v0.19.0 migrated the property 'labels' for processor")
}
