// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatormetrics

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewOperatorMetrics(t *testing.T) {
	config := &rest.Config{}
	scheme := runtime.NewScheme()
	metrics, err := NewOperatorMetrics(config, scheme, logr.Discard())
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
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, monitoringv1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "opentelemetry-operator", Namespace: "test-namespace", Labels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator", "control-plane": "controller-manager"}},
		},
	).Build()

	metrics := OperatorMetrics{kubeClient: client}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	go func() {
		errChan <- metrics.Start(ctx)
	}()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, time.Second*10)
	defer cancelTimeout()

	// Wait until one service monitor is being created
	var serviceMonitor *monitoringv1.ServiceMonitor = &monitoringv1.ServiceMonitor{}
	err = wait.PollUntilContextTimeout(
		ctxTimeout,
		time.Millisecond*100,
		time.Second*10,
		true,
		func(ctx context.Context) (bool, error) {
			errGet := client.Get(ctx, types.NamespacedName{Name: "opentelemetry-operator-metrics-monitor", Namespace: "test-namespace"}, serviceMonitor)

			if errGet != nil {
				if apierrors.IsNotFound(errGet) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		},
	)
	require.NoError(t, err)

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

func TestOperatorMetrics_getOwnerReferences(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		objects   []client.Object
		want      metav1.OwnerReference
		wantErr   bool
	}{
		{
			name:      "successful owner reference retrieval",
			namespace: "test-namespace",
			objects: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "opentelemetry-operator",
						Namespace: "test-namespace",
						UID:       "test-uid",
						Labels: map[string]string{
							"app.kubernetes.io/name": "opentelemetry-operator",
							"control-plane":          "controller-manager",
						},
					},
				},
			},
			want: metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "opentelemetry-operator",
				UID:        "test-uid",
			},
			wantErr: false,
		},
		{
			name:      "no deployments found",
			namespace: "test-namespace",
			objects:   []client.Object{},
			want:      metav1.OwnerReference{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			om := OperatorMetrics{
				kubeClient: fakeClient,
				log:        logr.Discard(),
			}

			got, err := om.getOwnerReferences(context.Background(), tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOwnerReferences() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOwnerReferences() got = %v, want %v", got, tt.want)
			}
		})
	}
}
