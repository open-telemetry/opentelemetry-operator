// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

func Test0_56_0Upgrade(t *testing.T) {
	one := int32(1)
	three := int32(3)

	collectorInstance := v1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "somewhere",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Replicas:    &one,
			MaxReplicas: &three,
			Config:      collectorCfg,
		},
	}

	collectorInstance.Status.Version = "0.55.0"
	versionUpgrade := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.56.0"),
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	upgradedInstanceV1beta1, err := versionUpgrade.ManagedInstance(context.Background(), convertTov1beta1(t, collectorInstance))
	assert.NoError(t, err)
	upgradedInstance := convertTov1alpha1(t, upgradedInstanceV1beta1)
	assert.Equal(t, one, *upgradedInstance.Spec.Autoscaler.MinReplicas)
}
