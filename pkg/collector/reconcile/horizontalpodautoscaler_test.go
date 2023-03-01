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

package reconcile

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var hpaUpdateErr error
var withHPA bool

func TestExpectedHPAVersionV2Beta2(t *testing.T) {
	params := paramsWithHPA(autodetect.AutoscalingVersionV2Beta2)
	err := params.Config.AutoDetect()
	assert.NoError(t, err)

	expectedHPA := collector.HorizontalPodAutoscaler(params.Config, logger, params.Instance)
	t.Run("should create HPA", func(t *testing.T) {
		err = expectedHorizontalPodAutoscalers(context.Background(), params, []client.Object{expectedHPA})
		assert.NoError(t, err)

		actual := autoscalingv2beta2.HorizontalPodAutoscaler{}
		exists, hpaErr := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		assert.NoError(t, hpaErr)
		require.Len(t, actual.Spec.Metrics, 1)
		assert.Equal(t, int32(90), *actual.Spec.Metrics[0].Resource.Target.AverageUtilization)

		assert.True(t, exists)
	})

	t.Run("should update HPA", func(t *testing.T) {
		minReplicas := int32(1)
		maxReplicas := int32(3)
		memUtilization := int32(70)
		updateParms := paramsWithHPA(autodetect.AutoscalingVersionV2Beta2)
		updateParms.Instance.Spec.Autoscaler.MinReplicas = &minReplicas
		updateParms.Instance.Spec.Autoscaler.MaxReplicas = &maxReplicas
		updateParms.Instance.Spec.Autoscaler.TargetMemoryUtilization = &memUtilization
		updatedHPA := collector.HorizontalPodAutoscaler(updateParms.Config, logger, updateParms.Instance)

		hpaUpdateErr = expectedHorizontalPodAutoscalers(context.Background(), updateParms, []client.Object{updatedHPA})
		require.NoError(t, hpaUpdateErr)

		actual := autoscalingv2beta2.HorizontalPodAutoscaler{}
		withHPA, hpaUpdateErr = populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, hpaUpdateErr)
		assert.True(t, withHPA)
		assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
		assert.Equal(t, int32(3), actual.Spec.MaxReplicas)
		assert.Len(t, actual.Spec.Metrics, 2)

		// check metric values
		for _, metric := range actual.Spec.Metrics {
			if metric.Resource.Name == corev1.ResourceCPU {
				assert.Equal(t, int32(90), *metric.Resource.Target.AverageUtilization)
			} else if metric.Resource.Name == corev1.ResourceMemory {
				assert.Equal(t, int32(70), *metric.Resource.Target.AverageUtilization)
			}
		}
	})

	t.Run("should delete HPA", func(t *testing.T) {
		err = deleteHorizontalPodAutoscalers(context.Background(), params, []client.Object{expectedHPA})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collecto"})
		assert.False(t, exists)
	})
}

func TestExpectedHPAVersionV2(t *testing.T) {
	params := paramsWithHPA(autodetect.AutoscalingVersionV2)
	err := params.Config.AutoDetect()
	assert.NoError(t, err)

	expectedHPA := collector.HorizontalPodAutoscaler(params.Config, logger, params.Instance)
	t.Run("should create HPA", func(t *testing.T) {
		err = expectedHorizontalPodAutoscalers(context.Background(), params, []client.Object{expectedHPA})
		assert.NoError(t, err)

		actual := autoscalingv2.HorizontalPodAutoscaler{}
		exists, hpaErr := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		assert.NoError(t, hpaErr)
		require.Len(t, actual.Spec.Metrics, 1)
		assert.Equal(t, int32(90), *actual.Spec.Metrics[0].Resource.Target.AverageUtilization)

		assert.True(t, exists)
	})

	t.Run("should update HPA", func(t *testing.T) {
		minReplicas := int32(1)
		maxReplicas := int32(3)
		memUtilization := int32(70)
		updateParms := paramsWithHPA(autodetect.AutoscalingVersionV2)
		updateParms.Instance.Spec.Autoscaler.MinReplicas = &minReplicas
		updateParms.Instance.Spec.Autoscaler.MaxReplicas = &maxReplicas
		updateParms.Instance.Spec.Autoscaler.TargetMemoryUtilization = &memUtilization
		updatedHPA := collector.HorizontalPodAutoscaler(updateParms.Config, logger, updateParms.Instance)

		hpaUpdateErr = expectedHorizontalPodAutoscalers(context.Background(), updateParms, []client.Object{updatedHPA})
		require.NoError(t, hpaUpdateErr)

		actual := autoscalingv2.HorizontalPodAutoscaler{}
		withHPA, hpaUpdateErr := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, hpaUpdateErr)
		assert.True(t, withHPA)
		assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
		assert.Equal(t, int32(3), actual.Spec.MaxReplicas)
		assert.Len(t, actual.Spec.Metrics, 2)
		// check metric values
		for _, metric := range actual.Spec.Metrics {
			if metric.Resource.Name == corev1.ResourceCPU {
				assert.Equal(t, int32(90), *metric.Resource.Target.AverageUtilization)
			} else if metric.Resource.Name == corev1.ResourceMemory {
				assert.Equal(t, int32(70), *metric.Resource.Target.AverageUtilization)
			}
		}
	})

	t.Run("should delete HPA", func(t *testing.T) {
		err = deleteHorizontalPodAutoscalers(context.Background(), params, []client.Object{expectedHPA})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collecto"})
		assert.False(t, exists)
	})
}

func paramsWithHPA(autoscalingVersion autodetect.AutoscalingVersion) Params {
	configYAML, err := os.ReadFile("../testdata/test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}

	minReplicas := int32(3)
	maxReplicas := int32(5)
	cpuUtilization := int32(90)

	mockAutoDetector := &mockAutoDetect{
		HPAVersionFunc: func() (autodetect.AutoscalingVersion, error) {
			return autoscalingVersion, nil
		},
	}
	configuration := config.New(config.WithAutoDetect(mockAutoDetector), config.WithCollectorImage(defaultCollectorImage), config.WithTargetAllocatorImage(defaultTaAllocationImage))
	err = configuration.AutoDetect()
	if err != nil {
		logger.Error(err, "configuration.autodetect failed")
	}

	return Params{
		Config: configuration,
		Client: k8sClient,
		Instance: v1alpha1.OpenTelemetryCollector{
			TypeMeta: metav1.TypeMeta{
				Kind:       "opentelemetry.io",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       instanceUID,
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Ports: []corev1.ServicePort{{
					Name: "web",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					NodePort: 0,
				}},
				Config: string(configYAML),
				Autoscaler: &v1alpha1.AutoscalerSpec{
					MinReplicas:          &minReplicas,
					MaxReplicas:          &maxReplicas,
					TargetCPUUtilization: &cpuUtilization,
				},
			},
		},
		Scheme:   testScheme,
		Log:      logger,
		Recorder: record.NewFakeRecorder(10),
	}
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	OpenShiftRoutesAvailabilityFunc func() (autodetect.OpenShiftRoutesAvailability, error)
	HPAVersionFunc                  func() (autodetect.AutoscalingVersion, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	return m.HPAVersionFunc()
}

func (m *mockAutoDetect) OpenShiftRoutesAvailability() (autodetect.OpenShiftRoutesAvailability, error) {
	if m.OpenShiftRoutesAvailabilityFunc != nil {
		return m.OpenShiftRoutesAvailabilityFunc()
	}
	return autodetect.OpenShiftRoutesNotAvailable, nil
}
