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

func Test0_38_0Upgrade(t *testing.T) {
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
				"--hii":         "hello",
				"--log-profile": "",
				"--log-format":  "hii",
				"--log-level":   "debug",
				"--arg1":        "",
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
	existing.Status.Version = "0.37.0"

	// TESTCASE 1: verify logging args exist and no config logging parameters
	// EXPECTED: drop logging args and configure logging parameters into config from args
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  version.Get(),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	res, err := up.ManagedInstance(context.Background(), existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, map[string]string{
		"--hii":  "hello",
		"--arg1": "",
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
    logs:
      development: true
      encoding: hii
      level: debug
`, res.Spec.Config)

	// TESTCASE 2: verify logging args exist and also config logging parameters exist
	// EXPECTED: drop logging args and persist logging parameters as configured in config
	configWithLogging := `exporters:
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
    logs:
      development: true
      encoding: hii
      level: debug
`
	existing.Spec.Config = configWithLogging
	existing.Spec.Args = map[string]string{
		"--hii":         "hello",
		"--log-profile": "",
		"--log-format":  "hii",
		"--log-level":   "debug",
		"--arg1":        "",
	}

	res, err = up.ManagedInstance(context.Background(), existing)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, configWithLogging, res.Spec.Config)
	assert.Equal(t, map[string]string{
		"--hii":  "hello",
		"--arg1": "",
	}, res.Spec.Args)
}
