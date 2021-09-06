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

func TestInfluxdbReceiverPropertyDrop(t *testing.T) {
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
			Config: `
receivers:
  influxdb:
    endpoint: 0.0.0.0:8080
    metrics_schema: telegraf-prometheus-v1

exporters:
  prometheusremotewrite:
    endpoint: "http:hello:4555/hii"

service:
  pipelines:
    metrics:
      receivers: [influxdb]
      exporters: [prometheusremotewrite]
`,
		},
	}
	existing.Status.Version = "0.30.0"

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, `exporters:
  prometheusremotewrite:
    endpoint: http:hello:4555/hii
receivers:
  influxdb:
    endpoint: 0.0.0.0:8080
service:
  pipelines:
    metrics:
      exporters:
      - prometheusremotewrite
      receivers:
      - influxdb
`, res.Spec.Config)
	assert.Equal(t, "upgrade to v0.31.0 dropped the 'metrics_schema' field from \"influxdb\" receiver", res.Status.Messages[0])
}
