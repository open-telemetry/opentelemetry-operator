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

func Test0_36_0Upgrade(t *testing.T) {
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
  otlp/mtls:
    protocols:
      grpc:
        endpoint: mysite.local:55690
        tls_settings:
          client_ca_file: client.pem
          cert_file: server.crt
          key_file: server.key
      http:
        endpoint: mysite.local:55690
        tls_settings:
          client_ca_file: client.pem
          cert_file: server.crt
          key_file: server.key
exporters:
  otlp:
    endpoint: "example.com"
    ca_file: /var/lib/mycert.pem
    insecure: true
    key_file: keyfile
    min_version: "1.0.0"
    max_version: "2.0.2"
    insecure_skip_verify: true
    server_name_override: hii

service:
  pipelines:
    traces:
      receivers: [otlp/mtls]
      exporters: [otlp]
`,
		},
	}
	existing.Status.Version = "0.35.0"

	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  version.Get(),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	// test
	resV1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)
	res := convertTov1alpha1(t, resV1beta1)

	// verify
	assert.YAMLEq(t, `exporters:
  otlp:
    endpoint: example.com
    tls:
      ca_file: /var/lib/mycert.pem
      insecure: true
      insecure_skip_verify: true
      key_file: keyfile
      max_version: 2.0.2
      min_version: 1.0.0
      server_name_override: hii
receivers:
  otlp/mtls:
    protocols:
      grpc:
        endpoint: mysite.local:55690
        tls:
          cert_file: server.crt
          client_ca_file: client.pem
          key_file: server.key
      http:
        endpoint: mysite.local:55690
        tls:
          cert_file: server.crt
          client_ca_file: client.pem
          key_file: server.key
service:
  pipelines:
    traces:
      exporters:
      - otlp
      processors: []
      receivers:
      - otlp/mtls
`, res.Spec.Config)
}
