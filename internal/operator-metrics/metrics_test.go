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

package operatormetrics

import (
	"context"
	"os"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewOperatorMetrics(t *testing.T) {
	config := &rest.Config{}
	scheme := runtime.NewScheme()
	metrics, err := NewOperatorMetrics(config, scheme)
	assert.NoError(t, err)
	assert.NotNil(t, metrics.kubeClient)
}

func TestOperatorMetrics_Start(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "namespace")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("test-namespace")
	require.NoError(t, err)
	tmpFile.Close()

	namespaceFile = tmpFile.Name()

	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = monitoringv1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	metrics := OperatorMetrics{kubeClient: client}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	go func() {
		errChan <- metrics.Start(ctx)
	}()

	// Wait a bit to allow the Start method to run
	time.Sleep(100 * time.Millisecond)

	cancel()
	err = <-errChan
	assert.NoError(t, err)
}

func TestOperatorMetrics_NeedLeaderElection(t *testing.T) {
	metrics := OperatorMetrics{}
	assert.True(t, metrics.NeedLeaderElection())
}

func TestOperatorMetrics_caConfigMapExists(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caBundleConfigMap,
				Namespace: openshiftInClusterMonitoringNamespace,
			},
		},
	).Build()

	metrics := OperatorMetrics{kubeClient: client}

	assert.True(t, metrics.caConfigMapExists())

	// Test when the ConfigMap doesn't exist
	clientWithoutConfigMap := fake.NewClientBuilder().WithScheme(scheme).Build()
	metricsWithoutConfigMap := OperatorMetrics{kubeClient: clientWithoutConfigMap}
	assert.False(t, metricsWithoutConfigMap.caConfigMapExists())
}
