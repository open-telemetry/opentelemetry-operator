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
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

func TestExpectedHPA(t *testing.T) {
	params := paramsWithHPA(autodetect.AutoscalingVersionV2Beta2)
	err := params.Config.AutoDetect()
	assert.NoError(t, err)
	autoscalingVersion := params.Config.AutoscalingVersion()

	expectedHPA := collector.HorizontalPodAutoscaler(params.Config, logger, params.Instance)
	t.Run("should create HPA", func(t *testing.T) {
		err = expectedHorizontalPodAutoscalers(context.Background(), params, []client.Object{expectedHPA})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &autoscalingv2beta2.HorizontalPodAutoscaler{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update HPA", func(t *testing.T) {
		minReplicas := int32(1)
		maxReplicas := int32(3)
		updateParms := paramsWithHPA(autodetect.AutoscalingVersionV2Beta2)
		updateParms.Instance.Spec.Replicas = &minReplicas
		updateParms.Instance.Spec.MaxReplicas = &maxReplicas
		updatedHPA := collector.HorizontalPodAutoscaler(updateParms.Config, logger, updateParms.Instance)

		if autoscalingVersion == autodetect.AutoscalingVersionV2Beta2 {
			updatedAutoscaler := *updatedHPA.(*autoscalingv2beta2.HorizontalPodAutoscaler)
			createObjectIfNotExists(t, "test-collector", &updatedAutoscaler)
			err := expectedHorizontalPodAutoscalers(context.Background(), updateParms, []client.Object{updatedHPA})
			assert.NoError(t, err)

			actual := autoscalingv2beta2.HorizontalPodAutoscaler{}
			exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

			assert.NoError(t, err)
			assert.True(t, exists)
			assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
			assert.Equal(t, int32(3), actual.Spec.MaxReplicas)
		} else {
			updatedAutoscaler := *updatedHPA.(*autoscalingv2.HorizontalPodAutoscaler)
			createObjectIfNotExists(t, "test-collector", &updatedAutoscaler)
			err := expectedHorizontalPodAutoscalers(context.Background(), updateParms, []client.Object{updatedHPA})
			assert.NoError(t, err)

			actual := autoscalingv2.HorizontalPodAutoscaler{}
			exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

			assert.NoError(t, err)
			assert.True(t, exists)
			assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
			assert.Equal(t, int32(3), actual.Spec.MaxReplicas)
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
				Config:      string(configYAML),
				Replicas:    &minReplicas,
				MaxReplicas: &maxReplicas,
				Autoscaler: &v1alpha1.AutoscalerSpec{
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
	PlatformFunc   func() (platform.Platform, error)
	HPAVersionFunc func() (autodetect.AutoscalingVersion, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	return m.HPAVersionFunc()
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
