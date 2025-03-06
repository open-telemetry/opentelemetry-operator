// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func TestRemoveQueuedRetryProcessor(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `
processors:
 queued_retry:
 otherprocessor:
 queued_retry/second:
   compression: "on"
   reconnection_delay: 15
   num_workers: 123

receivers:
  otlp: {}
exporters:
  debug: {}

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
`,
		},
	}
	existing.Status.Version = "0.18.0"

	// sanity check
	require.Contains(t, existing.Spec.Config, "queued_retry")
	require.Contains(t, existing.Spec.Config, "queued_retry/second")
	require.Contains(t, existing.Spec.Config, "num_workers: 123") // checking one property is sufficient

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.19.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	// verify
	assert.NotContains(t, res.Spec.Config, "queued_retry:")
	assert.Contains(t, res.Spec.Config, "otherprocessor:")
	assert.NotContains(t, res.Spec.Config, "queued_retry/second:")
	assert.NotContains(t, res.Spec.Config, "num_workers: 123") // checking one property is sufficient
}

func TestMigrateResourceType(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `
processors:
  resource:
    type: some-type

receivers:
  otlp: {}
exporters:
  debug: {}

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
`,
		},
	}
	existing.Status.Version = "0.18.0"

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.19.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	// verify
	assert.YAMLEq(t, `processors:
  resource:
    attributes:
    - action: upsert
      key: opencensus.type
      value: some-type

receivers:
  otlp: {}
exporters:
  debug: {}

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
`, res.Spec.Config)
}

func TestMigrateLabels(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `
processors:
  resource:
    labels:
      cloud.zone: zone-1
      host.name: k8s-node

receivers:
 otlp: {}
exporters:
 debug: {}

service:
 pipelines:
   traces:
     receivers: [otlp]
     exporters: [otlp]
     processors: [resource]
`,
		},
	}
	existing.Status.Version = "0.18.0"

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.19.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	actual, err := adapters.ConfigFromString(res.Spec.Config)
	require.NoError(t, err)
	actualProcessors := actual["processors"].(map[interface{}]interface{})
	actualProcessor := actualProcessors["resource"].(map[interface{}]interface{})
	actualAttrs := actualProcessor["attributes"].([]interface{})

	// verify
	assert.Len(t, actualAttrs, 2)
	assert.Nil(t, actualProcessor["labels"])
}
