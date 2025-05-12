// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_41_0Upgrade(t *testing.T) {
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
receivers:
 otlp:
   cors_allowed_origins:
   - https://foo.bar.com
   - https://*.test.com
   cors_allowed_headers:
   - ExampleHeader
exporters:
  debug: {}

service:
 pipelines:
   metrics:
     receivers: [otlp]
     exporters: [debug]
`,
		},
	}
	existing.Status.Version = "0.40.0"

	// TESTCASE 1: restructure cors for both allowed_origin & allowed_headers
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.41.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	assert.YAMLEq(t, `receivers:
  otlp:
    cors:
      allowed_headers:
      - ExampleHeader
      allowed_origins:
      - https://foo.bar.com
      - https://*.test.com
exporters:
  debug: {}
service:
  pipelines:
    metrics:
      exporters:
      - debug
      receivers:
      - otlp
`, res.Spec.Config)

	// TESTCASE 2: re-structure cors for allowed_origins
	existing = v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `
receivers:
 otlp:
   cors_allowed_origins:
   - https://foo.bar.com
   - https://*.test.com

exporters:
  debug: {}

service:
 pipelines:
   metrics:
     receivers: [otlp]
     exporters: [debug]
`,
		},
	}

	existing.Status.Version = "0.40.0"
	resV1beta1, err = up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res = convertTov1alpha1(t, resV1beta1)

	assert.YAMLEq(t, `receivers:
  otlp:
    cors:
      allowed_origins:
      - https://foo.bar.com
      - https://*.test.com
exporters:
  debug: {}
service:
  pipelines:
    metrics:
      exporters:
      - debug
      receivers:
      - otlp
`, res.Spec.Config)
}
