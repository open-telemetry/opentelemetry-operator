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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

var logger = logf.Log.WithName("unit-tests")

func TestShouldUpgradeAllToLatestBasedOnUpgradeStrategy(t *testing.T) {
	const beginV = "0.0.1" // this is the first version we have an upgrade function

	currentV := version.Get()
	currentV.OpenTelemetryCollector = upgrade.Latest.String()

	for _, tt := range []struct {
		strategy  v1alpha1.UpgradeStrategy
		expectedV string
	}{
		{v1alpha1.UpgradeStrategyAutomatic, upgrade.Latest.String()},
		{v1alpha1.UpgradeStrategyNone, beginV},
	} {
		t.Run("spec.UpgradeStrategy = "+string(tt.strategy), func(t *testing.T) {
			// prepare
			nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
			existing := makeOtelcol(nsn, v1alpha1.ManagementStateManaged)
			err := k8sClient.Create(context.Background(), &existing)
			require.NoError(t, err)

			existing.Status.Version = beginV
			err = k8sClient.Status().Update(context.Background(), &existing)
			require.NoError(t, err)

			// sanity check
			persisted := &v1alpha1.OpenTelemetryCollector{}
			err = k8sClient.Get(context.Background(), nsn, persisted)
			require.NoError(t, err)
			require.Equal(t, beginV, persisted.Status.Version)
			up := &upgrade.VersionUpgrade{
				Log:      logger,
				Version:  currentV,
				Client:   k8sClient,
				Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
			}

			// test
			err = up.ManagedInstances(context.Background())
			assert.NoError(t, err)

			// verify
			err = k8sClient.Get(context.Background(), nsn, persisted)
			assert.NoError(t, err)
			assert.Equal(t, upgrade.Latest.String(), persisted.Status.Version)

			// cleanup
			assert.NoError(t, k8sClient.Delete(context.Background(), &existing))
		})
	}
}

func TestEnvVarUpdates(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	collectorInstance := v1beta1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
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
						"prometheus": nil,
					},
				},
				Exporters: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"debug": []interface{}{},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {
							Exporters:  []string{"debug"},
							Processors: nil,
							Receivers:  []string{"prometheus"},
						},
					},
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), &collectorInstance)
	require.NoError(t, err)

	collectorInstance.Status.Version = "0.104.0"
	err = k8sClient.Status().Update(context.Background(), &collectorInstance)
	require.NoError(t, err)
	// sanity check
	persisted := &v1beta1.OpenTelemetryCollector{}
	err = k8sClient.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	require.Equal(t, collectorInstance.Status.Version, persisted.Status.Version)

	currentV := version.Get()
	currentV.OpenTelemetryCollector = "0.111.0"
	up := &upgrade.VersionUpgrade{
		Log:      logger,
		Version:  currentV,
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
	}

	// test
	err = up.ManagedInstances(context.Background())
	assert.NoError(t, err)

	// verify
	err = k8sClient.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, upgrade.Latest.String(), persisted.Status.Version)
	assert.NotContainsf(t, persisted.Spec.Args["feature-gates"], "-confmap.unifyEnvVarExpansion", "still has env var")

	// cleanup
	assert.NoError(t, k8sClient.Delete(context.Background(), &collectorInstance))
}

func TestUpgradeUpToLatestKnownVersion(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		v         string
		expectedV string
	}{
		{"upgrade-routine", "0.8.0", "0.10.0"},     // we don't have a 0.10.0 upgrade, but we have a 0.9.0
		{"no-upgrade-routine", "0.61.1", "0.62.0"}, // No upgrade routines between these two versions
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
			existing := makeOtelcol(nsn, v1alpha1.ManagementStateManaged)
			existing.Status.Version = tt.v

			currentV := version.Get()
			currentV.OpenTelemetryCollector = tt.expectedV
			up := &upgrade.VersionUpgrade{
				Log:      logger,
				Version:  currentV,
				Client:   k8sClient,
				Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
			}
			// test
			res, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))

			// verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedV, res.Status.Version)
		})
	}
}

func TestVersionsShouldNotBeChanged(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	for _, tt := range []struct {
		desc            string
		v               string
		expectedV       string
		failureExpected bool
		managementState v1alpha1.ManagementStateType
	}{
		{"new-instance", "", "", false, v1alpha1.ManagementStateManaged},
		{"newer-than-our-newest", "100.0.0", "100.0.0", false, v1alpha1.ManagementStateManaged},
		{"unparseable", "unparseable", "unparseable", true, v1alpha1.ManagementStateManaged},
		// Ignore unmanaged instances
		{"unmanaged-instance", "1.0.0", "1.0.0", false, v1alpha1.ManagementStateUnmanaged},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			existing := makeOtelcol(nsn, tt.managementState)
			existing.Status.Version = tt.v

			currentV := version.Get()
			currentV.OpenTelemetryCollector = upgrade.Latest.String()

			up := &upgrade.VersionUpgrade{
				Log:      logger,
				Version:  currentV,
				Client:   k8sClient,
				Recorder: record.NewFakeRecorder(upgrade.RecordBufferSize),
			}

			// test
			res, err := up.ManagedInstance(context.Background(), convertTov1beta1(t, existing))
			if tt.failureExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// verify
			assert.Equal(t, tt.expectedV, res.Status.Version)
		})
	}
}

const collectorCfg = `---
receivers:
  otlp:
    protocols:
      grpc: {}
processors:
  batch: {}
exporters:
  otlp:
    endpoint: "otlp:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
`

func makeOtelcol(nsn types.NamespacedName, managementState v1alpha1.ManagementStateType) v1alpha1.OpenTelemetryCollector {
	return v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			ManagementState: managementState,
			Config:          collectorCfg,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
	}
}

func convertTov1beta1(t *testing.T, collector v1alpha1.OpenTelemetryCollector) v1beta1.OpenTelemetryCollector {
	betacollector := v1beta1.OpenTelemetryCollector{}
	err := collector.ConvertTo(&betacollector)
	require.NoError(t, err)
	return betacollector
}

func convertTov1alpha1(t *testing.T, collector v1beta1.OpenTelemetryCollector) v1alpha1.OpenTelemetryCollector {
	alphacollector := v1alpha1.OpenTelemetryCollector{}
	err := alphacollector.ConvertFrom(&collector)
	require.NoError(t, err)
	return alphacollector
}
