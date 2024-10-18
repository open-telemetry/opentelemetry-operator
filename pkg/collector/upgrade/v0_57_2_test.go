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
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_57_0Upgrade(t *testing.T) {
	collectorInstance := v1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-my-instance",
			Namespace: "somewhere",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `receivers:
  otlp:
    protocols:
      http:
        endpoint: mysite.local:55690
extensions:
  health_check:
    endpoint: "localhost"
    port: "4444"
    check_collector_pipeline:
      enabled: false
      exporter_failure_threshold: 5
      interval: 5m
exporters:
  debug: {}
service:
  extensions: [health_check]
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [debug]
`,
		},
	}

	collectorInstance.Status.Version = "0.56.0"
	//Test to remove port and change endpoint value.
	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.57.2"),
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}

	upgradedInstanceV1beta1, err := versionUpgrade.ManagedInstance(context.Background(), convertTov1beta1(t, collectorInstance))
	assert.NoError(t, err)
	upgradedInstance := convertTov1alpha1(t, upgradedInstanceV1beta1)
	assert.YAMLEq(t, `extensions:
  health_check:
    check_collector_pipeline:
      enabled: false
      exporter_failure_threshold: 5
      interval: 5m
    endpoint: localhost:4444
receivers:
  otlp:
    protocols:
      http:
        endpoint: mysite.local:55690
exporters:
  debug: {}
service:
  extensions:
  - health_check
  pipelines:
    metrics:
      exporters:
      - debug
      receivers:
      - otlp
`, upgradedInstance.Spec.Config)
}
