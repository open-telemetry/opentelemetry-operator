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
      exporters: [otlp]
`,
		},
	}
	existing.Status.Version = "0.42.0"

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, map[string]string{
		"--test-upgrade43": "true",
		"--test-arg1":      "otel",
	}, res.Spec.Args)

	// verify
	assert.Equal(t, `exporters:
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
      receivers:
      - otlp/mtls
  telemetry:
    metrics:
      address: :8988
      level: detailed
`, res.Spec.Config)

	assert.Equal(t, "upgrade to v0.43.0 dropped the deprecated metrics arguments "+"i.e. [--metrics-addr --metrics-level] from otelcol custom resource otelcol.spec.args and "+"adding them to otelcol.spec.config.service.telemetry.metrics, if no metrics arguments are configured already.", res.Status.Messages[0])

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
	res, err = upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, configWithMetrics, res.Spec.Config)
	assert.Equal(t, map[string]string{
		"--test-upgrade43": "true",
		"--test-arg1":      "otel",
	}, res.Spec.Args)

	assert.Equal(t, "upgrade to v0.43.0 dropped the deprecated metrics arguments "+"i.e. [--metrics-addr --metrics-level] from otelcol custom resource otelcol.spec.args and "+"adding them to otelcol.spec.config.service.telemetry.metrics, if no metrics arguments are configured already.", res.Status.Messages[0])
}
