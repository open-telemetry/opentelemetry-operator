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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/version"
	"github.com/signalfx/splunk-otel-operator/pkg/collector/upgrade"
)

var logger = logf.Log.WithName("unit-tests")

func TestShouldUpgradeAllToLatest(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.SplunkOtelAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "splunk-otel-operator",
			},
		},
	}
	existing.Status.Version = "0.0.1" // this is the first version we have an upgrade function
	err := k8sClient.Create(context.Background(), &existing)
	require.NoError(t, err)

	err = k8sClient.Status().Update(context.Background(), &existing)
	require.NoError(t, err)

	currentV := version.Get()
	currentV.SplunkOtelAgent = upgrade.Latest.String()

	// sanity check
	persisted := &v1alpha1.SplunkOtelAgent{}
	err = k8sClient.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	require.Equal(t, "0.0.1", persisted.Status.Version)

	// test
	err = upgrade.ManagedInstances(context.Background(), logger, currentV, k8sClient)
	assert.NoError(t, err)

	// verify
	err = k8sClient.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, upgrade.Latest.String(), persisted.Status.Version)

	// cleanup
	assert.NoError(t, k8sClient.Delete(context.Background(), &existing))
}

func TestUpgradeUpToLatestKnownVersion(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	existing := v1alpha1.SplunkOtelAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "splunk-otel-operator",
			},
		},
	}
	existing.Status.Version = "0.8.0"

	currentV := version.Get()
	currentV.SplunkOtelAgent = "0.10.0" // we don't have a 0.10.0 upgrade, but we have a 0.9.0

	// test
	res, err := upgrade.ManagedInstance(context.Background(), logger, currentV, k8sClient, existing)

	// verify
	assert.NoError(t, err)
	assert.Equal(t, "0.10.0", res.Status.Version)
}

func TestVersionsShouldNotBeChanged(t *testing.T) {
	for _, tt := range []struct {
		desc            string
		v               string
		expectedV       string
		failureExpected bool
	}{
		{"new-instance", "", "", false},
		{"newer-than-our-newest", "100.0.0", "100.0.0", false},
		{"unparseable", "unparseable", "unparseable", true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
			existing := v1alpha1.SplunkOtelAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nsn.Name,
					Namespace: nsn.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "splunk-otel-operator",
					},
				},
			}
			existing.Status.Version = tt.v

			currentV := version.Get()
			currentV.SplunkOtelAgent = upgrade.Latest.String()

			// test
			res, err := upgrade.ManagedInstance(context.Background(), logger, currentV, k8sClient, existing)
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
