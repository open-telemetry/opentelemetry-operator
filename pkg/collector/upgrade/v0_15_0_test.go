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

func TestRemoveMetricsTypeFlags(t *testing.T) {
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
				// this would not happen in the real world, as it's either one or another, but we aren't going that far
				"--new-metrics":    "true",
				"--legacy-metrics": "true",
			},
		},
	}
	existing.Status.Version = "0.9.0"

	// sanity check
	require.Contains(t, existing.Spec.Args, "--new-metrics")
	require.Contains(t, existing.Spec.Args, "--legacy-metrics")

	// test
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  makeVersion("0.15.0"),
		Client:   nil,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}
	res, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
	assert.NoError(t, err)

	// verify
	assert.NotContains(t, res.Spec.Args, "--new-metrics")
	assert.NotContains(t, res.Spec.Args, "--legacy-metrics")
}
