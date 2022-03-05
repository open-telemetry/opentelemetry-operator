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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestIsHPARequired(t *testing.T) {
	for _, tt := range []struct {
		params   Params
		required bool
	}{
		{paramsWithHPA(), true},
		{params(), false},
	} {
		r := isHPARequired(tt.params)
		assert.Equal(t, r, tt.required)
	}
}

func TestExpectedHPA(t *testing.T) {
	params := paramsWithHPA()
	expectedHPA := collector.HorizontalPodAutoscaler(params.Config, logger, params.Instance)

	t.Run("should create HPA", func(t *testing.T) {
		err := expectedHorizontalPodAutoscalers(context.Background(), params, []autoscalingv1.HorizontalPodAutoscaler{expectedHPA})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &autoscalingv1.HorizontalPodAutoscaler{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should update HPA", func(t *testing.T) {
		minReplicas := int32(1)
		maxReplicas := int32(3)
		updateParms := paramsWithHPA()
		updateParms.Instance.Spec.Replicas = &minReplicas
		updateParms.Instance.Spec.MaxReplicas = &maxReplicas
		updatedHPA := collector.HorizontalPodAutoscaler(updateParms.Config, logger, updateParms.Instance)

		createObjectIfNotExists(t, "test-collector", &updatedHPA)
		err := expectedHorizontalPodAutoscalers(context.Background(), updateParms, []autoscalingv1.HorizontalPodAutoscaler{updatedHPA})
		assert.NoError(t, err)

		actual := autoscalingv1.HorizontalPodAutoscaler{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
		assert.Equal(t, int32(3), actual.Spec.MaxReplicas)
	})

	t.Run("should delete HPA", func(t *testing.T) {
		err := deleteHorizontalPodAutoscalers(context.Background(), params, []autoscalingv1.HorizontalPodAutoscaler{expectedHPA})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collecto"})
		assert.False(t, exists)
	})
}

func paramsWithHPA() Params {
	configYAML, err := ioutil.ReadFile("../testdata/test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}

	enabled := true
	minReplicas := int32(3)
	maxReplicas := int32(5)

	return Params{
		Config: config.New(config.WithCollectorImage(defaultCollectorImage), config.WithTargetAllocatorImage(defaultTaAllocationImage)),
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
				Autoscale:   &enabled,
				Replicas:    &minReplicas,
				MaxReplicas: &maxReplicas,
			},
		},
		Scheme:   testScheme,
		Log:      logger,
		Recorder: record.NewFakeRecorder(10),
	}
}
