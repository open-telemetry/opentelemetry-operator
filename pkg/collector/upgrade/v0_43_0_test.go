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
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_43_0Upgrade(t *testing.T) {
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
			Args: map[string]string{
				"--metrics-addr":   ":8988",
				"--metrics-level":  "detailed",
				"--test-upgrade43": "true",
				"--test-arg1":      "otel",
			},
			Config: `
receivers:
  otlp/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690

exporters:
  otlp:
    endpoint: "example.com"

service:
  pipelines:
    traces: 
      receivers: [otlp/mtls]
      processors: []
      exporters: [otlp]
`,
		},
	}
	existing.Status.Version = "0.42.0"

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  version.Get(),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	// verify
	assert.Equal(t, map[string]string{
		"--test-upgrade43": "true",
		"--test-arg1":      "otel",
	}, res.Spec.Args)

	// verify
	assert.YAMLEq(t, `exporters:
  otlp:
    endpoint: example.com
receivers:
  otlp/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690
service:
  pipelines:
    traces:
      exporters:
      - otlp
      processors: []
      receivers:
      - otlp/mtls
  telemetry:
    metrics:
      address: :8988
      level: detailed
`, res.Spec.Config)

	configWithMetrics := `exporters:
  otlp:
    endpoint: example.com
receivers:
  otlp/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690
service:
  pipelines:
    traces:
      exporters:
      - otlp
      processors: []
      receivers:
      - otlp/mtls
  telemetry:
    metrics:
      address: :8988
      level: detailed
`
	existing.Spec.Config = configWithMetrics
	existing.Spec.Args = map[string]string{
		"--metrics-addr":   ":8988",
		"--metrics-level":  "detailed",
		"--test-upgrade43": "true",
		"--test-arg1":      "otel",
	}
	resV1beta1, err = up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res = convertTov1alpha1(t, resV1beta1)

	// verify
	assert.YAMLEq(t, configWithMetrics, res.Spec.Config)
	assert.Equal(t, map[string]string{
		"--test-upgrade43": "true",
		"--test-arg1":      "otel",
	}, res.Spec.Args)

}
