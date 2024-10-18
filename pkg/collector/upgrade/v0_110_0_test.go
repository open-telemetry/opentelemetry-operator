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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func Test0_110_0Upgrade(t *testing.T) {
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
			Version: "0.104.0",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Args: map[string]string{
					"foo":           "bar",
					"feature-gates": "+baz,-component.UseLocalHostAsDefaultHost",
				},
			},
			Config: v1beta1.Config{},
		},
	}

	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.110.0"),
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}

	col, err := versionUpgrade.ManagedInstance(context.Background(), collectorInstance)
	if err != nil {
		t.Errorf("expect err: nil but got: %v", err)
	}
	assert.EqualValues(t,
		map[string]string{"foo": "bar", "feature-gates": "+baz"}, col.Spec.Args)
}
