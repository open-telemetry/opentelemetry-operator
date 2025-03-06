// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

func Test0_105_0Upgrade(t *testing.T) {
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
					"feature-gates": "+baz,-confmap.unifyEnvVarExpansion",
				},
			},
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": []interface{}{},
					},
				},
			},
		},
	}

	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.105.0"),
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

func TestRemoveFeatureGate(t *testing.T) {
	tests := []struct {
		test     string
		args     map[string]string
		feature  string
		expected map[string]string
	}{
		{
			test:     "empty",
			args:     map[string]string{},
			expected: map[string]string{},
		},
		{
			test:     "no feature gates",
			args:     map[string]string{"foo": "bar"},
			feature:  "foo",
			expected: map[string]string{"foo": "bar"},
		},
		{
			test:     "remove enabled feature gate",
			args:     map[string]string{"foo": "bar", "feature-gates": "+foo"},
			feature:  "-foo",
			expected: map[string]string{"foo": "bar", "feature-gates": "+foo"},
		},
		{
			test:     "remove disabled feature gate",
			args:     map[string]string{"foo": "bar", "feature-gates": "-foo"},
			feature:  "-foo",
			expected: map[string]string{"foo": "bar"},
		},
		{
			test:     "remove disabled feature gate, start",
			args:     map[string]string{"foo": "bar", "feature-gates": "-foo,bar"},
			feature:  "-foo",
			expected: map[string]string{"foo": "bar", "feature-gates": "bar"},
		},
		{
			test:     "remove disabled feature gate, end",
			args:     map[string]string{"foo": "bar", "feature-gates": "bar,-foo"},
			feature:  "-foo",
			expected: map[string]string{"foo": "bar", "feature-gates": "bar"},
		},
	}

	for _, test := range tests {
		t.Run(test.test, func(t *testing.T) {
			args := upgrade.RemoveFeatureGate(test.args, test.feature)
			assert.Equal(t, test.expected, args)
		})
	}
}
