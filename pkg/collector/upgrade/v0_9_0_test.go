// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func TestRemoveConnectionDelay(t *testing.T) {
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
			Config: `exporters:
  opencensus:
    compression: "on"
    reconnection_delay: 15
    num_workers: 123`,
		},
	}
	existing.Status.Version = "0.8.0"

	// sanity check
	require.Contains(t, existing.Spec.Config, "reconnection_delay")

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.9.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	resv1beta1, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)

	res := convertTov1alpha1(t, resv1beta1)
	// verify
	assert.Contains(t, res.Spec.Config, "opencensus:")
	assert.Contains(t, res.Spec.Config, `compression: "on"`)
	assert.NotContains(t, res.Spec.Config, "reconnection_delay")
	assert.Contains(t, res.Spec.Config, "num_workers: 123")
}
