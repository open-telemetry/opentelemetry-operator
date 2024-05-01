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

func Test0_39_0Upgrade(t *testing.T) {
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
  httpd/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690

  httpd:

processors:
  memory_limiter:
  memory_limiter/with-settings:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
    ballast_size_mib: 2000

exporters:
  debug: {}

service:
  pipelines:
    metrics:
      receivers: [httpd/mtls, httpd]
      exporters: [debug]
`,
		},
	}
	existing.Status.Version = "0.38.0"

	// TESTCASE 1: verify httpd receiver renamed to apache
	// drop processors.memory_limiter field 'ballast_size_mib'
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  version.Get(),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	assert.YAMLEq(t, `processors:
  memory_limiter:
  memory_limiter/with-settings:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
receivers:
  apache:
  apache/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690
exporters:
  debug: {}
service:
  pipelines:
    metrics:
      exporters:
      - debug
      processors: []
      receivers:
      - apache/mtls
      - apache
`, res.Spec.Config)

	// TESTCASE 2: Drop ballast_size_mib from memory_limiter processor
	existing1 := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `receivers:
  otlp/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690

  otlp:

processors:
  memory_limiter:
  memory_limiter/with-settings:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
    ballast_size_mib: 2000

exporters:
  debug: {}

service:
  pipelines:
    traces:
      receivers: [otlp/mtls, otlp]
      exporters: [debug]
`,
		},
	}

	existing1.Status.Version = "0.38.0"
	resV1beta1, err = up.ManagedInstance(context.Background(), convertTov1beta1(t, existing1))
	assert.NoError(t, err)
	res = convertTov1alpha1(t, resV1beta1)

	// verify
	assert.YAMLEq(t, `processors:
  memory_limiter:
  memory_limiter/with-settings:
    check_interval: 5s
    limit_mib: 4000
    spike_limit_mib: 500
receivers:
  otlp:
  otlp/mtls:
    protocols:
      http:
        endpoint: mysite.local:55690

exporters:
  debug: {}

service:
  pipelines:
    traces:
      exporters:
      - debug
      processors: []
      receivers:
      - otlp/mtls
      - otlp
`, res.Spec.Config)
}
