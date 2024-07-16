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
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_104_0Upgrade(t *testing.T) {

	collectorInstance := v1beta1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-my-instance",
			Namespace: "somewhere",
		},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			Version: "0.103.0",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"otlp": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": nil,
								"http": nil,
							},
						},
						"otlp/nothing": &v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"protocols": nil,
							},
						},
						"otlp/empty": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": map[string]interface{}{
									"endpoint": "",
								},
								"http": map[string]interface{}{
									"endpoint": "",
								},
							},
						},
						"otlp/something": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": map[string]interface{}{
									"endpoint": "123.123.123.123:8642",
								},
								"http": map[string]interface{}{
									"endpoint": "123.123.123.123:8642",
								},
							},
						},
					},
				},
			},
		},
	}

	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  version.Get(),
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}

	_, err := versionUpgrade.ManagedInstance(context.Background(), collectorInstance)
	if err != nil {
		t.Errorf("expect err: nil but got: %v", err)
	}

	assert.EqualValues(t, map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{
				"endpoint": "0.0.0.0:4317",
			},
			"http": map[string]interface{}{
				"endpoint": "0.0.0.0:4318",
			},
		},
	}, collectorInstance.Spec.Config.Receivers.Object["otlp"], "normal entry is not up-to-date")

	assert.EqualValues(t, &v1beta1.AnyConfig{
		Object: map[string]interface{}{
			"protocols": nil,
		},
	}, collectorInstance.Spec.Config.Receivers.Object["otlp/nothing"], "no updated expected")

	assert.EqualValues(t, map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{
				"endpoint": "0.0.0.0:4317",
			},
			"http": map[string]interface{}{
				"endpoint": "0.0.0.0:4318",
			},
		},
	}, collectorInstance.Spec.Config.Receivers.Object["otlp/empty"], "empty entry is not up-to-date")

	assert.EqualValues(t, map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{
				"endpoint": "123.123.123.123:8642",
			},
			"http": map[string]interface{}{
				"endpoint": "123.123.123.123:8642",
			},
		},
	}, collectorInstance.Spec.Config.Receivers.Object["otlp/something"], "endpoints exist, did not expect an update")

}
