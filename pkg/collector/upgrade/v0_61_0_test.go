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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

var (
	//go:embed testdata/v0_61_0-valid.yaml
	valid string
	//go:embed testdata/v0_61_0-invalid.yaml
	invalid string
)

func Test0_61_0Upgrade(t *testing.T) {

	collectorInstance := v1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-my-instance",
			Namespace: "somewhere",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	tt := []struct {
		name      string
		config    string
		expectErr bool
	}{
		{
			name:      "no remote sampling config", // valid
			config:    valid,
			expectErr: false,
		},
		{
			name:      "has remote sampling config", // invalid
			config:    invalid,
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			collectorInstance.Spec.Config = tc.config
			collectorInstance.Status.Version = "0.60.0"

			versionUpgrade := &upgrade.VersionUpgrade{
				Log:      logger,
				Version:  makeVersion("0.61.0"),
				Client:   k8sClient,
				Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
			}

			_, err := versionUpgrade.ManagedInstance(context.Background(), convertTov1beta1(t, collectorInstance))
			if (err != nil) != tc.expectErr {
				t.Errorf("expect err: %t but got: %v", tc.expectErr, err)
			}
		})
	}
}
